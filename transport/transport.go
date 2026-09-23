// Package transport turns the `tls` fragment of a service's configuration
// into the two TLS configurations a service needs: one for the listener it
// serves, one for the connections it makes.
//
// It does three things, and deliberately no more:
//
//  1. loads the certificate the platform mounted, and RELOADS it when the
//     platform replaces it;
//  2. presents it, as a server and as a client;
//  3. admits a peer by the ACCOUNT it runs as, read from the identity in its
//     certificate, against a list the configuration gives.
//
// It does not fetch a certificate, mint one, or talk to an authority. That is
// the platform's job, and a service that did it would be asserting an identity
// rather than presenting one it was given.
package transport

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// Mode says which listeners a service serves.
type Mode string

const (
	// Off is cleartext only. It is the default, and it is what a service
	// installed on a platform that provides no identity must fall back to.
	Off Mode = "off"

	// Permissive serves both, on two ports, so one edge can migrate at a
	// time. It is a migration state and should carry a date.
	Permissive Mode = "permissive"

	// Strict serves only the authenticated listener.
	Strict Mode = "strict"
)

// Peer is an account that may call this service.
type Peer struct {
	Namespace      string `json:"namespace"`
	ServiceAccount string `json:"serviceAccount"`
}

// Config is the `tls` fragment, decoded.
type Config struct {
	Mode        Mode   `json:"mode"`
	Address     string `json:"address"`
	CertFile    string `json:"certFile"`
	KeyFile     string `json:"keyFile"`
	CAFile      string `json:"caFile"`
	TrustDomain string `json:"trustDomain"`
	Peers       []Peer `json:"peers"`
}

// Identity is a loaded workload identity.
type Identity struct {
	cfg   Config
	roots *x509.CertPool
	log   *slog.Logger

	// The certificate, and what it was loaded from. A rotation replaces the
	// files in place, so the modification time is what says it is stale.
	mu       sync.RWMutex
	cert     *tls.Certificate
	loadedAt time.Time
}

// Load reads the mounted identity. It returns nil when the mode is off,
// which callers treat as "serve cleartext" rather than as an error: a chart
// whose default is off must produce a service that runs.
//
// The logger is where REFUSALS go, and passing one is not optional in
// spirit. A caller is told only that it was refused — telling it which rule
// rejected it describes the allow-list to whoever is probing — so the
// server's log is the only place the reason exists. A service that refuses
// everyone for an unrelated reason and a service that is working look
// identical without it. A nil logger is accepted for a test and discards
// them.
func Load(cfg Config, log *slog.Logger) (*Identity, error) {
	if cfg.Mode == "" {
		cfg.Mode = Off
	}

	switch cfg.Mode {
	case Off:
		return nil, nil
	case Permissive, Strict:
	default:
		return nil, fmt.Errorf("transport mode %q is not off, permissive or strict", cfg.Mode)
	}

	if cfg.Mode == Permissive && cfg.Address == "" {
		return nil, fmt.Errorf("permissive needs an address for the authenticated listener: one listener cannot be both")
	}

	if cfg.TrustDomain == "" {
		return nil, fmt.Errorf("no trust domain, so every peer's identity would be admitted from anywhere")
	}

	pem, err := os.ReadFile(cfg.CAFile)
	if err != nil {
		return nil, fmt.Errorf("read the trust bundle: %w", err)
	}

	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("the trust bundle at %s holds no certificate", cfg.CAFile)
	}

	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}

	id := &Identity{cfg: cfg, roots: roots, log: log}
	if _, err := id.certificate(); err != nil {
		return nil, err
	}

	return id, nil
}

// Mode is what the configuration asked for; Off when nothing was loaded.
func (i *Identity) Mode() Mode {
	if i == nil {
		return Off
	}

	return i.cfg.Mode
}

// certificate returns the current certificate, re-reading it when the files
// on disk have changed.
//
// Re-reading rather than holding what start-up loaded is the whole point. A
// platform rotates these on an hour or less; a process that cached the first
// one authenticates fine until that expires and then fails everywhere at
// once, with an error about an expired certificate and nothing pointing at
// the line that read it.
func (i *Identity) certificate() (*tls.Certificate, error) {
	info, err := os.Stat(i.cfg.CertFile)
	if err != nil {
		return nil, fmt.Errorf("stat the certificate: %w", err)
	}

	i.mu.RLock()
	cached, at := i.cert, i.loadedAt
	i.mu.RUnlock()

	if cached != nil && !info.ModTime().After(at) {
		return cached, nil
	}

	pair, err := tls.LoadX509KeyPair(i.cfg.CertFile, i.cfg.KeyFile)
	if err != nil {
		// A rotation is not atomic across two files, so a read that
		// catches it half-done fails here. The previous certificate is
		// still valid, so serving it beats refusing the connection.
		if cached != nil {
			return cached, nil
		}

		return nil, fmt.Errorf("load the certificate and its key: %w", err)
	}

	i.mu.Lock()
	i.cert, i.loadedAt = &pair, info.ModTime()
	i.mu.Unlock()

	return &pair, nil
}

// Server is the configuration for this service's own listener: it presents
// the mounted certificate, requires one from every caller, and admits only
// the accounts the configuration named.
func (i *Identity) Server() *tls.Config {
	if i == nil {
		return nil
	}

	return &tls.Config{
		MinVersion: tls.VersionTLS13,
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			return i.certificate()
		},
		// The chain is verified by the library against the trust bundle;
		// WHO the verified peer is, is this package's question.
		ClientAuth:            tls.RequireAndVerifyClientCert,
		ClientCAs:             i.roots,
		VerifyPeerCertificate: i.verifyPeer,
	}
}

// Client is the configuration for connections this service makes: it
// presents the same certificate and checks the server's IDENTITY.
//
// It does not check the server's name, and `InsecureSkipVerify` is how that
// is expressed — which reads alarming and is not, because the chain and the
// identity are both verified below instead. The name is skipped because a
// platform's workload certificates carry no name: they carry an identity,
// which is a stronger statement. "I reached the address I meant to" is a
// weaker question than "the thing answering is the account I was told to
// trust", and the second one is the one being asked.
//
// The library's own verification has to be turned off to ask it, because it
// insists on the name. So this does the library's job by hand: build the
// chain against the trust bundle, then read the identity out of the leaf.
// Skipping either half would be the mistake the flag's name warns about.
func (i *Identity) Client() *tls.Config {
	if i == nil {
		return nil
	}

	return &tls.Config{
		MinVersion: tls.VersionTLS13,
		GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return i.certificate()
		},
		// codeql[go/disabled-certificate-check] The certificate IS checked,
		// by verifyChainAndPeer below: it builds the chain against the trust
		// bundle and then reads the peer's identity. Only the NAME check is
		// skipped, because a platform's workload certificates carry no name.
		// Two tests hold the property this flag would otherwise destroy:
		// TestAClientRefusesAServerOutsideItsTrustBundle and
		// TestAClientRefusesAServerItWasNotToldToTrust.
		//nolint:gosec // see above
		InsecureSkipVerify:    true,
		VerifyPeerCertificate: i.verifyChainAndPeer,
	}
}

// verifyChainAndPeer does what the library would have done, plus the part it
// cannot: verify the chain against the trust bundle, then check the peer's
// identity. It is used where the name check had to be turned off.
func (i *Identity) verifyChainAndPeer(rawCerts [][]byte, _ [][]*x509.Certificate) error {
	if len(rawCerts) == 0 {
		return fmt.Errorf("the peer presented no certificate")
	}

	certs := make([]*x509.Certificate, 0, len(rawCerts))

	for _, raw := range rawCerts {
		cert, err := x509.ParseCertificate(raw)
		if err != nil {
			return fmt.Errorf("parse the peer's certificate: %w", err)
		}

		certs = append(certs, cert)
	}

	intermediates := x509.NewCertPool()
	for _, cert := range certs[1:] {
		intermediates.AddCert(cert)
	}

	chains, err := certs[0].Verify(x509.VerifyOptions{
		Roots:         i.roots,
		Intermediates: intermediates,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	})
	if err != nil {
		return fmt.Errorf("the peer's certificate does not chain to the trust bundle: %w", err)
	}

	return i.verifyPeer(nil, chains)
}

// verifyPeer runs after the library has verified the chain. It reads the
// peer's identity out of the leaf and checks it against the allow-list.
func (i *Identity) verifyPeer(_ [][]byte, chains [][]*x509.Certificate) error {
	if len(chains) == 0 || len(chains[0]) == 0 {
		return fmt.Errorf("the peer presented no verified certificate")
	}

	peer, err := i.identityOf(chains[0][0])
	if err != nil {
		return err
	}

	for _, allowed := range i.cfg.Peers {
		if allowed == peer {
			return nil
		}
	}

	// Said HERE, because the caller will not be told. The refusal it
	// receives is a bare alert; this line is the only record that the
	// service made a decision rather than breaking.
	// Background, because there is none: this runs inside a TLS handshake,
	// before any request exists to carry one.
	i.log.WarnContext(context.Background(), "refused a peer",
		slog.String("namespace", peer.Namespace),
		slog.String("serviceAccount", peer.ServiceAccount))

	return fmt.Errorf("%s/%s is not a peer this service admits", peer.Namespace, peer.ServiceAccount)
}

// identityOf reads the account out of a certificate's identity URI.
//
// The shape is the SPIFFE one: spiffe://<trust domain>/ns/<namespace>/sa/<account>.
// Anything else is refused rather than guessed at, because a partial match
// here is an identity nobody meant to grant.
func (i *Identity) identityOf(leaf *x509.Certificate) (Peer, error) {
	for _, u := range leaf.URIs {
		if u.Scheme != "spiffe" {
			continue
		}

		if u.Host != i.cfg.TrustDomain {
			return Peer{}, fmt.Errorf("the peer's identity belongs to trust domain %q, not %q", u.Host, i.cfg.TrustDomain)
		}

		return parsePath(u)
	}

	return Peer{}, fmt.Errorf("the peer's certificate carries no identity")
}

func parsePath(u *url.URL) (Peer, error) {
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) != 4 || parts[0] != "ns" || parts[2] != "sa" {
		return Peer{}, fmt.Errorf("the peer's identity %q does not name a namespace and an account", u.String())
	}

	if parts[1] == "" || parts[3] == "" {
		return Peer{}, fmt.Errorf("the peer's identity %q names an empty namespace or account", u.String())
	}

	return Peer{Namespace: parts[1], ServiceAccount: parts[3]}, nil
}
