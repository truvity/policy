package transport_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/truvity/policy/transport"
)

const trustDomain = "example.internal"

// authority is a throwaway certificate authority standing in for the
// platform's. The point of the tests below is what this package does with
// what it is handed, so the handing is the cheap part.
type authority struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
	dir  string
}

func newAuthority(t *testing.T) *authority {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	must(t, err)

	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test authority"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	must(t, err)

	cert, err := x509.ParseCertificate(der)
	must(t, err)

	a := &authority{cert: cert, key: key, dir: t.TempDir()}

	must(t, os.WriteFile(filepath.Join(a.dir, "ca.pem"),
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600))

	return a
}

// issue writes a certificate for one account, the way the platform's driver
// mounts one, and returns the configuration a service holding it would read.
func (a *authority) issue(t *testing.T, name, namespace, account string, peers ...transport.Peer) transport.Config {
	t.Helper()

	dir := filepath.Join(a.dir, name)
	must(t, os.MkdirAll(dir, 0o700))
	a.write(t, dir, namespace, account)

	return transport.Config{
		Mode:        transport.Strict,
		CertFile:    filepath.Join(dir, "tls.crt"),
		KeyFile:     filepath.Join(dir, "tls.key"),
		CAFile:      filepath.Join(a.dir, "ca.pem"),
		TrustDomain: trustDomain,
		Peers:       peers,
	}
}

func (a *authority) write(t *testing.T, dir, namespace, account string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	must(t, err)

	id := &url.URL{
		Scheme: "spiffe",
		Host:   trustDomain,
		Path:   fmt.Sprintf("/ns/%s/sa/%s", namespace, account),
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: account},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		URIs:         []*url.URL{id},
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, a.cert, &key.PublicKey, a.key)
	must(t, err)

	keyDER, err := x509.MarshalECPrivateKey(key)
	must(t, err)

	must(t, os.WriteFile(filepath.Join(dir, "tls.crt"),
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600))
	must(t, os.WriteFile(filepath.Join(dir, "tls.key"),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600))
}

// serve stands up a real TLS listener with the given identity and returns
// its URL and a client presenting its own.
//
// Not httptest's TLS helper: that one substitutes its own certificate when
// the configuration carries none in `Certificates`, and this package
// deliberately supplies GetCertificate instead — so the helper would test
// its own certificate rather than the one under test.
func serve(t *testing.T, server, client *transport.Identity) (string, *http.Client, *serverLog) {
	t.Helper()

	ln, err := tls.Listen("tcp", "127.0.0.1:0", server.Server())
	must(t, err)

	// The server's own errors, captured rather than printed. A refusal's
	// REASON stays on the server deliberately — the caller is told only
	// that it was refused — so this is the only place a test can read it.
	errs := &serverLog{}

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, "served")
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ErrorLog:          log.New(errs, "", 0),
	}

	go func() { _ = srv.Serve(ln) }()

	t.Cleanup(func() { _ = srv.Close() })

	return "https://" + ln.Addr().String(), &http.Client{
		Transport: &http.Transport{TLSClientConfig: client.Client()},
	}, errs
}

// serverLog collects what the server wrote, safely enough for a test that
// reads it from a different goroutine than the one that wrote it.
type serverLog struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (l *serverLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.buf.Write(p)
}

func (l *serverLog) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.buf.String()
}

// Off is the default, and it must produce a service that RUNS. A chart whose
// default is off is installed by someone whose platform provides none of
// this, and an error here would be a chart nobody outside can use.
func TestOffLoadsNothingAndIsNotAnError(t *testing.T) {
	id, err := transport.Load(transport.Config{}, nil)
	must(t, err)
	mustBeNil(t, id, "expected nil")
	if id.Mode() != transport.Off {
		t.Errorf("got %v, want %v", id.Mode(), transport.Off)
	}
	mustBeNil(t, id.Server(), "a nil identity serves cleartext")
	mustBeNil(t, id.Client(), "expected nil")
}

// The happy path, end to end over a real handshake: both sides present a
// mounted identity and each admits the other's account.
func TestAnAdmittedPeerIsServed(t *testing.T) {
	ca := newAuthority(t)

	server, err := transport.Load(ca.issue(t, "server", "shop", "api",
		transport.Peer{Namespace: "shop", ServiceAccount: "web"}), nil)
	must(t, err)

	client, err := transport.Load(ca.issue(t, "client", "shop", "web",
		transport.Peer{Namespace: "shop", ServiceAccount: "api"}), nil)
	must(t, err)

	url, httpClient, _ := serve(t, server, client)

	resp, err := httpClient.Get(url)
	must(t, err)

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	must(t, err)
	if string(body) != "served" {
		t.Errorf("got %q, want %q", body, "served")
	}
}

// THE negative test. A caller with a valid certificate from the right
// authority, whose account is not on the list, is refused at the handshake —
// not at the application, not with a 403, but before a byte of the request
// is read.
func TestAPeerNotOnTheListIsClosedAtTheHandshake(t *testing.T) {
	ca := newAuthority(t)

	refusals := &serverLog{}

	server, err := transport.Load(ca.issue(t, "server", "shop", "api",
		transport.Peer{Namespace: "shop", ServiceAccount: "web"}),
		slog.New(slog.NewJSONHandler(refusals, nil)))
	must(t, err)

	// A real workload, properly issued, simply not granted.
	stranger, err := transport.Load(ca.issue(t, "stranger", "shop", "batch",
		transport.Peer{Namespace: "shop", ServiceAccount: "api"}), nil)
	must(t, err)

	url, httpClient, serverErrs := serve(t, server, stranger)

	_, err = httpClient.Get(url)
	mustFail(t, err)

	// The CALLER is told only that it was refused. Leaking which rule
	// rejected it would tell an attacker what the allow-list contains.
	mustFailWith(t, err, "tls:")

	// The SERVER knows why, and says so where an operator will read it.
	// Without this half, a service that refused everyone for an unrelated
	// reason would pass this test.
	if !strings.Contains(serverErrs.String(), "shop/batch is not a peer this service admits") {
		t.Errorf("the server did not say why it refused; it logged: %s", serverErrs.String())
	}

	// And it says so through the service's OWN logger, which is where an
	// operator looks — not only in whatever the HTTP server happens to
	// print. A refusal nobody can find is a refusal nobody can tell from a
	// service that is simply broken.
	if !strings.Contains(refusals.String(), `"serviceAccount":"batch"`) {
		t.Errorf("the refusal did not reach the service's logger; it holds: %s", refusals.String())
	}
}

// A certificate from another trust domain is refused even if its account
// matches an entry, because the account name alone is not the identity.
func TestAnIdentityFromAnotherTrustDomainIsRefused(t *testing.T) {
	ours, theirs := newAuthority(t), newAuthority(t)

	server, err := transport.Load(ours.issue(t, "server", "shop", "api",
		transport.Peer{Namespace: "shop", ServiceAccount: "web"}), nil)
	must(t, err)

	// Same namespace, same account, different authority: the chain check
	// catches this one before the identity check does, which is the order
	// it should be caught in.
	other := theirs.issue(t, "client", "shop", "web",
		transport.Peer{Namespace: "shop", ServiceAccount: "api"})

	client, err := transport.Load(other, nil)
	must(t, err)

	url, httpClient, _ := serve(t, server, client)

	_, err = httpClient.Get(url)
	mustFail(t, err)
}

// A rotation must be picked up without a restart. This is the failure that
// only appears an hour after a deploy, which is long enough that nobody
// connects it to the deploy.
func TestARotatedCertificateIsPickedUpWithoutRestarting(t *testing.T) {
	ca := newAuthority(t)
	cfg := ca.issue(t, "server", "shop", "api", transport.Peer{Namespace: "shop", ServiceAccount: "web"})

	id, err := transport.Load(cfg, nil)
	must(t, err)

	first, err := id.Server().GetCertificate(&tls.ClientHelloInfo{})
	must(t, err)

	// The platform replaces the files in place. A modification time in the
	// future stands in for the wait, because the filesystem's resolution is
	// coarser than this test is fast.
	dir := filepath.Dir(cfg.CertFile)
	ca.write(t, dir, "shop", "api")
	future := time.Now().Add(time.Second)
	must(t, os.Chtimes(cfg.CertFile, future, future))

	second, err := id.Server().GetCertificate(&tls.ClientHelloInfo{})
	must(t, err)

	if bytes.Equal(first.Certificate[0], second.Certificate[0]) {
		t.Error("the certificate on disk changed and the process is still serving the old one")
	}
}

// A half-written rotation must not take the listener down: the previous
// certificate is still valid, so serving it beats refusing every connection
// for the moment it takes the platform to finish writing.
func TestAHalfWrittenRotationKeepsServing(t *testing.T) {
	ca := newAuthority(t)
	cfg := ca.issue(t, "server", "shop", "api", transport.Peer{Namespace: "shop", ServiceAccount: "web"})

	id, err := transport.Load(cfg, nil)
	must(t, err)

	_, err = id.Server().GetCertificate(&tls.ClientHelloInfo{})
	must(t, err)

	// The certificate has been replaced; the key has not yet.
	must(t, os.WriteFile(cfg.CertFile, []byte("-----BEGIN CERTIFICATE-----\ntorn\n"), 0o600))
	future := time.Now().Add(time.Second)
	must(t, os.Chtimes(cfg.CertFile, future, future))

	got, err := id.Server().GetCertificate(&tls.ClientHelloInfo{})
	must(t, err)
	if got == nil {
		t.Fatal("expected a certificate")
	}
}

// Refusals at load time, each naming what is missing rather than failing
// later and further away.
func TestLoadRefuses(t *testing.T) {
	ca := newAuthority(t)

	t.Run("an unknown mode", func(t *testing.T) {
		_, err := transport.Load(transport.Config{Mode: "mutual"}, nil)
		mustFailWith(t, err, "not off, permissive or strict")
	})

	t.Run("no trust domain", func(t *testing.T) {
		cfg := ca.issue(t, "s1", "shop", "api")
		cfg.TrustDomain = ""
		_, err := transport.Load(cfg, nil)
		mustFailWith(t, err, "no trust domain")
	})

	t.Run("a trust bundle that is not one", func(t *testing.T) {
		cfg := ca.issue(t, "s2", "shop", "api")
		cfg.CAFile = filepath.Join(t.TempDir(), "empty.pem")
		must(t, os.WriteFile(cfg.CAFile, []byte("not a certificate"), 0o600))
		_, err := transport.Load(cfg, nil)
		mustFailWith(t, err, "holds no certificate")
	})

	t.Run("a certificate that is not there", func(t *testing.T) {
		cfg := ca.issue(t, "s3", "shop", "api")
		cfg.CertFile = filepath.Join(t.TempDir(), "absent.crt")
		_, err := transport.Load(cfg, nil)
		mustFail(t, err)
	})
}

// An empty peer list admits nobody. That is the right default for a service
// nobody has been granted, and it must fail closed rather than open.
func TestAnEmptyPeerListAdmitsNobody(t *testing.T) {
	ca := newAuthority(t)

	server, err := transport.Load(ca.issue(t, "server", "shop", "api"), nil)
	must(t, err)

	client, err := transport.Load(ca.issue(t, "client", "shop", "web",
		transport.Peer{Namespace: "shop", ServiceAccount: "api"}), nil)
	must(t, err)

	url, httpClient, _ := serve(t, server, client)

	_, err = httpClient.Get(url)
	mustFail(t, err)
}

// The helpers this package's tests use instead of an assertion library. The
// root module depends on three packages and nothing else, which is a
// property worth more to whoever imports it than a terser test is worth
// here.

func must(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatal(err)
	}
}

func mustFail(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("expected an error, got none")
	}
}

func mustFailWith(t *testing.T, err error, want string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected an error mentioning %q, got none", want)
	}

	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error %q does not mention %q", err, want)
	}
}

func mustBeNil(t *testing.T, v any, why string) {
	t.Helper()

	if v != nil && !reflect.ValueOf(v).IsNil() {
		t.Fatalf("%s: got %v", why, v)
	}
}

// The client turns the library's verification off so that it can ignore the
// server's NAME — a platform's certificates carry none. This proves it does
// not ignore anything else: a server whose certificate does not chain to the
// client's trust bundle is refused by the client, at the handshake.
//
// It is the test that justifies the flag. Without it, "we verify by hand"
// is a claim rather than a property, and the flag means what its name says.
func TestAClientRefusesAServerOutsideItsTrustBundle(t *testing.T) {
	ours, theirs := newAuthority(t), newAuthority(t)

	// A server with a perfectly good certificate — from the wrong
	// authority, and naming an account the client would otherwise admit.
	server, err := transport.Load(theirs.issue(t, "server", "shop", "api",
		transport.Peer{Namespace: "shop", ServiceAccount: "web"}), nil)
	must(t, err)

	client, err := transport.Load(ours.issue(t, "client", "shop", "web",
		transport.Peer{Namespace: "shop", ServiceAccount: "api"}), nil)
	must(t, err)

	url, httpClient, _ := serve(t, server, client)

	_, err = httpClient.Get(url)
	mustFail(t, err)
	mustFailWith(t, err, "does not chain to the trust bundle")
}

// And the identity is checked on the client's side too: a server that chains
// correctly but runs as an account the client was not told to trust is
// refused. Together with the test above, the two halves the library would
// have done are both accounted for.
func TestAClientRefusesAServerItWasNotToldToTrust(t *testing.T) {
	ca := newAuthority(t)

	server, err := transport.Load(ca.issue(t, "server", "shop", "somebody-else",
		transport.Peer{Namespace: "shop", ServiceAccount: "web"}), nil)
	must(t, err)

	client, err := transport.Load(ca.issue(t, "client", "shop", "web",
		transport.Peer{Namespace: "shop", ServiceAccount: "api"}), nil)
	must(t, err)

	url, httpClient, _ := serve(t, server, client)

	_, err = httpClient.Get(url)
	mustFail(t, err)
	mustFailWith(t, err, "shop/somebody-else is not a peer this service admits")
}
