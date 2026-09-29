package transport_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/truvity/policy/transport"
)

// broker stands in for a server that is not a workload: a NAME, a
// certificate from its own chain, and a demand that every client present a
// certificate from the workload identity chain. It reports the identity URI
// each verified client presented.
type broker struct {
	addr    string
	name    string
	pool    *x509.CertPool
	clients chan string
}

func newBroker(t *testing.T, name string, clientCA *authority) *broker {
	t.Helper()

	chain := newAuthority(t)

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	must(t, err)

	der, err := x509.CreateCertificate(rand.Reader, &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: "broker"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{name},
	}, chain.cert, &key.PublicKey, chain.key)
	must(t, err)

	clientPool := x509.NewCertPool()
	clientPool.AddCert(clientCA.cert)

	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientPool,
	})
	must(t, err)

	b := &broker{addr: ln.Addr().String(), name: name, clients: make(chan string, 16)}

	b.pool = x509.NewCertPool()
	b.pool.AddCert(chain.cert)

	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}

			go func() {
				defer func() { _ = conn.Close() }()

				tc, ok := conn.(*tls.Conn)
				if !ok || tc.Handshake() != nil {
					return
				}

				b.clients <- tc.ConnectionState().PeerCertificates[0].URIs[0].String()

				// Echo until the client goes away: the connection under
				// test outlives a rotation.
				buf := make([]byte, 16)
				for {
					n, err := tc.Read(buf)
					if err != nil {
						return
					}

					if _, err := tc.Write(buf[:n]); err != nil {
						return
					}
				}
			}()
		}
	}()

	return b
}

func (b *broker) saw(t *testing.T) string {
	t.Helper()

	select {
	case who := <-b.clients:
		return who
	case <-time.After(5 * time.Second):
		t.Fatal("the broker never saw a verified client")

		return ""
	}
}

// The client presents the workload identity to a server that has a name and
// no identity, and verifies that server the ordinary way.
func TestClientToPresentsTheIdentityAndVerifiesTheServerByName(t *testing.T) {
	ca := newAuthority(t)
	b := newBroker(t, "broker.example", ca)

	id, err := transport.Load(ca.issue(t, "client", "shop", "web"), nil)
	must(t, err)

	conn, err := tls.Dial("tcp", b.addr, id.ClientTo(b.pool, b.name))
	must(t, err)

	defer func() { _ = conn.Close() }()

	if got, want := b.saw(t), "spiffe://"+trustDomain+"/ns/shop/sa/web"; got != want {
		t.Errorf("the broker saw %q, want %q", got, want)
	}
}

// Rotation, the property the platform depends on: a connection already made
// is NOT dropped when the files are replaced, and the NEXT connect presents
// the new certificate without a restart.
func TestClientToPicksUpARotationOnTheNextConnectAndDropsNothing(t *testing.T) {
	ca := newAuthority(t)
	b := newBroker(t, "broker.example", ca)

	cfg := ca.issue(t, "client", "shop", "web")
	id, err := transport.Load(cfg, nil)
	must(t, err)

	first, err := tls.Dial("tcp", b.addr, id.ClientTo(b.pool, b.name))
	must(t, err)

	defer func() { _ = first.Close() }()

	b.saw(t)

	firstSerial := serialPresented(t, id)

	// The platform replaces the files in place.
	ca.write(t, filepath.Dir(cfg.CertFile), "shop", "web")

	future := time.Now().Add(time.Second)
	must(t, os.Chtimes(cfg.CertFile, future, future))

	// The connection made before the rotation still works.
	if _, err := first.Write([]byte("ping")); err != nil {
		t.Fatalf("the rotation dropped a connection already made: %v", err)
	}

	echo := make([]byte, 4)
	if _, err := first.Read(echo); err != nil || string(echo) != "ping" {
		t.Fatalf("the connection made before the rotation stopped answering: %q, %v", echo, err)
	}

	// The next connect presents the new certificate.
	second, err := tls.Dial("tcp", b.addr, id.ClientTo(b.pool, b.name))
	must(t, err)

	defer func() { _ = second.Close() }()

	b.saw(t)

	if serialPresented(t, id).Cmp(firstSerial) == 0 {
		t.Error("the certificate on disk changed and the next connect still presented the old one")
	}
}

func serialPresented(t *testing.T, id *transport.Identity) *big.Int {
	t.Helper()

	got, err := id.ClientTo(nil, "").GetClientCertificate(&tls.CertificateRequestInfo{})
	must(t, err)

	leaf, err := x509.ParseCertificate(got.Certificate[0])
	must(t, err)

	return leaf.SerialNumber
}

// The broker's name and chain are checked: a certificate for another name,
// or from a chain the client was not given, is refused.
func TestClientToRefusesAServerWithTheWrongNameOrChain(t *testing.T) {
	ca := newAuthority(t)
	b := newBroker(t, "broker.example", ca)

	id, err := transport.Load(ca.issue(t, "client", "shop", "web"), nil)
	must(t, err)

	if _, err := tls.Dial("tcp", b.addr, id.ClientTo(b.pool, "someone-else.example")); err == nil {
		t.Error("a server certificate for another name was accepted")
	}

	if _, err := tls.Dial("tcp", b.addr, id.ClientTo(x509.NewCertPool(), b.name)); err == nil {
		t.Error("a server certificate from a chain the client was not given was accepted")
	}
}

// With no identity there is nothing to present, and the answer is nil rather
// than a TLS configuration that dials without a client certificate.
func TestClientToWithNoIdentityIsNil(t *testing.T) {
	var id *transport.Identity

	if cfg := id.ClientTo(x509.NewCertPool(), "x"); cfg != nil {
		t.Errorf("no identity produced a TLS configuration: %v", cfg)
	}

}
