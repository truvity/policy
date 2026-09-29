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

	a.writeShaped(t, dir, namespace, account, nil)
}

// writeShaped is write with a hook that may bend the certificate before it is
// signed: the tests below use it to mint what a correct platform never would.
func (a *authority) writeShaped(t *testing.T, dir, namespace, account string, bend func(*x509.Certificate)) {
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

	if bend != nil {
		bend(tmpl)
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
// its URL, a client presenting its own, the server's own captured errors,
// and a channel that receives a value each time one of the server's
// connections reaches net/http's StateClosed.
//
// Not httptest's TLS helper: that one substitutes its own certificate when
// the configuration carries none in `Certificates`, and this package
// deliberately supplies GetCertificate instead — so the helper would test
// its own certificate rather than the one under test.
//
// The closed channel exists because a connection is handled entirely by its
// own goroutine (net/http's own per-connection `serve`, spawned from
// `Serve`): on a handshake failure that goroutine writes to `errs` — via
// the Server's own `ErrorLog` — and only THEN, as that goroutine unwinds,
// does net/http mark the connection StateClosed and invoke `ConnState`. A
// test that waits for StateClosed is therefore reading `errs` only after
// the write it depends on has already happened in that other goroutine,
// rather than racing it — which reading `errs` right after the CLIENT's
// request merely failed does not guarantee: the client only needs the TLS
// alert the library sends BEFORE it ever gets around to logging the
// handshake failure, so under load the client can observe the refusal and
// a test can go on to read `errs` before the server has written to it.
func serve(t *testing.T, server, client *transport.Identity) (string, *http.Client, *serverLog, <-chan struct{}) {
	t.Helper()

	ln, err := tls.Listen("tcp", "127.0.0.1:0", server.Server())
	must(t, err)

	// The server's own errors, captured rather than printed. A refusal's
	// REASON stays on the server deliberately — the caller is told only
	// that it was refused — so this is the only place a test can read it.
	errs := &serverLog{}

	// Buffered generously rather than blocking: nothing here needs every
	// transition delivered, only that at least one StateClosed arrives
	// after the one connection each of these tests drives, and a hook that
	// blocked on a full channel would wedge the connection's own goroutine
	// (and so net/http's shutdown) on tests that never read it at all.
	closed := make(chan struct{}, 16)

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, "served")
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ErrorLog:          log.New(errs, "", 0),
		ConnState: func(_ net.Conn, state http.ConnState) {
			if state == http.StateClosed {
				select {
				case closed <- struct{}{}:
				default:
				}
			}
		},
	}

	go func() { _ = srv.Serve(ln) }()

	t.Cleanup(func() { _ = srv.Close() })

	return "https://" + ln.Addr().String(), &http.Client{
		Transport: &http.Transport{TLSClientConfig: client.Client()},
	}, errs, closed
}

// waitClosed blocks until serve's ConnState hook has observed a connection
// reach http.StateClosed — the deterministic point, explained on serve's
// own doc comment, after which it is safe to read what the server logged
// about that connection. The bound is a safety net for a genuine hang, not
// a timing guess: the transition it waits for follows immediately behind
// the write it is ordering against, in the same goroutine.
func waitClosed(t *testing.T, closed <-chan struct{}) {
	t.Helper()
	select {
	case <-closed:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for the server to close the connection")
	}
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

	url, httpClient, _, _ := serve(t, server, client)

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

	url, httpClient, serverErrs, closed := serve(t, server, stranger)

	_, err = httpClient.Get(url)
	mustFail(t, err)

	// The CALLER is told only that it was refused. Leaking which rule
	// rejected it would tell an attacker what the allow-list contains.
	mustFailWith(t, err, "tls:")

	// The client above only needed the TLS alert the server sends BEFORE
	// it logs anything — see serve's own doc comment — so reading
	// serverErrs here without waiting for the connection to actually close
	// would race the server's own goroutine under load. This makes the
	// ordering the two checks below depend on explicit instead.
	waitClosed(t, closed)

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

	url, httpClient, _, _ := serve(t, server, client)

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

	url, httpClient, _, _ := serve(t, server, client)

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

	url, httpClient, _, _ := serve(t, server, client)

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

	url, httpClient, _, _ := serve(t, server, client)

	_, err = httpClient.Get(url)
	mustFail(t, err)
	mustFailWith(t, err, "shop/somebody-else is not a peer this service admits")
}

// issueShaped is issue for a certificate the platform would never mint.
func (a *authority) issueShaped(t *testing.T, name string, bend func(*x509.Certificate), peers ...transport.Peer) transport.Config {
	t.Helper()

	cfg := a.issue(t, name, "shop", "api", peers...)
	a.writeShaped(t, filepath.Dir(cfg.CertFile), "shop", "api", bend)

	return cfg
}

// An X509-SVID leaf is not a certificate authority, and may not sign
// certificates or revocation lists. The peer here is the very account the
// server admits, with a chain that verifies; only its shape is wrong.
func TestAPeerLeafThatIsAnAuthorityOrMaySignIsRefused(t *testing.T) {
	cases := map[string]struct {
		bend func(*x509.Certificate)
		want string
	}{
		"a leaf that is a CA": {
			bend: func(c *x509.Certificate) { c.IsCA, c.BasicConstraintsValid = true, true },
			want: "is a certificate authority",
		},
		"a leaf that may sign certificates": {
			bend: func(c *x509.Certificate) { c.KeyUsage |= x509.KeyUsageCertSign },
			want: "may sign certificates",
		},
		"a leaf that may sign revocation lists": {
			bend: func(c *x509.Certificate) { c.KeyUsage |= x509.KeyUsageCRLSign },
			want: "may sign certificates",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ca := newAuthority(t)

			server, err := transport.Load(ca.issue(t, "server", "shop", "web",
				transport.Peer{Namespace: "shop", ServiceAccount: "api"}), nil)
			must(t, err)

			// Client side of the check: the server presents the bent leaf.
			client, err := transport.Load(ca.issue(t, "client", "shop", "api",
				transport.Peer{Namespace: "shop", ServiceAccount: "api"}), nil)
			must(t, err)

			// Server side of the check: the client presents the bent leaf.
			bentClient, err := transport.Load(ca.issueShaped(t, "bent-client", tc.bend,
				transport.Peer{Namespace: "shop", ServiceAccount: "web"}), nil)
			must(t, err)

			bentServer, err := transport.Load(ca.issueShaped(t, "bent-server", tc.bend,
				transport.Peer{Namespace: "shop", ServiceAccount: "api"}), nil)
			must(t, err)

			// The client refuses the bent server.
			url, httpClient, _, _ := serve(t, bentServer, client)
			_, err = httpClient.Get(url)
			mustFail(t, err)
			mustFailWith(t, err, tc.want)

			// The server refuses the bent client.
			url, httpClient, errs, closed := serve(t, server, bentClient)
			_, err = httpClient.Get(url)
			mustFail(t, err)
			waitClosed(t, closed)

			if !strings.Contains(errs.String(), tc.want) {
				t.Fatalf("the server did not say why: %q", errs.String())
			}
		})
	}
}

// An X509-SVID carries exactly one URI name, of any scheme: with two there is
// no telling which is the identity, so the certificate is refused whole.
func TestAPeerLeafWithMoreOrFewerThanOneURIIsRefused(t *testing.T) {
	other := func(raw string) *url.URL {
		u, err := url.Parse(raw)
		must(t, err)

		return u
	}

	cases := map[string]struct {
		bend func(*x509.Certificate)
		want string
	}{
		"two spiffe IDs": {
			bend: func(c *x509.Certificate) {
				c.URIs = append(c.URIs, other("spiffe://"+trustDomain+"/ns/shop/sa/second"))
			},
			want: "2 URI names",
		},
		"a spiffe ID and another scheme": {
			bend: func(c *x509.Certificate) { c.URIs = append(c.URIs, other("https://example.org/other")) },
			want: "2 URI names",
		},
		"no URI at all": {
			bend: func(c *x509.Certificate) { c.URIs = nil },
			want: "carries no identity",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ca := newAuthority(t)

			server, err := transport.Load(ca.issue(t, "server", "shop", "web",
				transport.Peer{Namespace: "shop", ServiceAccount: "api"}), nil)
			must(t, err)

			client, err := transport.Load(ca.issue(t, "client", "shop", "api",
				transport.Peer{Namespace: "shop", ServiceAccount: "api"}), nil)
			must(t, err)

			bentClient, err := transport.Load(ca.issueShaped(t, "bent-client", tc.bend,
				transport.Peer{Namespace: "shop", ServiceAccount: "web"}), nil)
			must(t, err)

			bentServer, err := transport.Load(ca.issueShaped(t, "bent-server", tc.bend,
				transport.Peer{Namespace: "shop", ServiceAccount: "api"}), nil)
			must(t, err)

			url, httpClient, _, _ := serve(t, bentServer, client)
			_, err = httpClient.Get(url)
			mustFail(t, err)
			mustFailWith(t, err, tc.want)

			url, httpClient, errs, closed := serve(t, server, bentClient)
			_, err = httpClient.Get(url)
			mustFail(t, err)
			waitClosed(t, closed)

			if !strings.Contains(errs.String(), tc.want) {
				t.Fatalf("the server did not say why: %q", errs.String())
			}
		})
	}
}
