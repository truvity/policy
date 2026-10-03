// Code generated from Pkl module `spike.contract.urlshortener.Web`. DO NOT EDIT.
package web

import (
	"context"

	"github.com/apple/pkl-go/pkl"
	"spike.invalid/pklconf/gen/fragments"
	"spike.invalid/pklconf/gen/service"
)

type Web interface {
	service.Service

	GetListen() fragments.Listen

	GetTls() *fragments.TlsFields

	GetUrls() Urls

	GetCsp() *Csp

	GetFaro() *Faro

	GetAssets() Assets
}

var _ Web = WebImpl{}

// The front end: serve the page, and ask the service that owns the tables. It writes nothing and holds no database credential.
type WebImpl struct {
	service.ServiceImpl

	Listen fragments.Listen `pkl:"listen"`

	Tls *fragments.TlsFields `pkl:"tls"`

	// The service that owns the URL tables. An address and nothing else: which protocol the caller speaks is in its code, and whether the connection is authenticated is the tls block's business.
	Urls Urls `pkl:"urls"`

	// The Content-Security-Policy this server sends with every response. Absent means report-only with no extra origins: the header is sent, nothing is blocked, so turning it on cannot break a page.
	Csp *Csp `pkl:"csp"`

	// Browser telemetry (Grafana Faro). Absent, or `enabled: false`, means the page sends nothing and the Faro libraries are never downloaded. Every field here is written into the page for every visitor to read: `apiKey` is a PUBLIC identifier of this app at the collector (one key per app, rotatable, paired with the collector's origin allow-list), not a secret, and no credential may be put in this block.
	Faro *Faro `pkl:"faro"`

	// Where the built page is. A path rather than an embedded bundle, because the assets are the one thing in this image that a CDN might serve instead.
	Assets Assets `pkl:"assets"`
}

func (rcv WebImpl) GetListen() fragments.Listen {
	return rcv.Listen
}

func (rcv WebImpl) GetTls() *fragments.TlsFields {
	return rcv.Tls
}

// The service that owns the URL tables. An address and nothing else: which protocol the caller speaks is in its code, and whether the connection is authenticated is the tls block's business.
func (rcv WebImpl) GetUrls() Urls {
	return rcv.Urls
}

// The Content-Security-Policy this server sends with every response. Absent means report-only with no extra origins: the header is sent, nothing is blocked, so turning it on cannot break a page.
func (rcv WebImpl) GetCsp() *Csp {
	return rcv.Csp
}

// Browser telemetry (Grafana Faro). Absent, or `enabled: false`, means the page sends nothing and the Faro libraries are never downloaded. Every field here is written into the page for every visitor to read: `apiKey` is a PUBLIC identifier of this app at the collector (one key per app, rotatable, paired with the collector's origin allow-list), not a secret, and no credential may be put in this block.
func (rcv WebImpl) GetFaro() *Faro {
	return rcv.Faro
}

// Where the built page is. A path rather than an embedded bundle, because the assets are the one thing in this image that a CDN might serve instead.
func (rcv WebImpl) GetAssets() Assets {
	return rcv.Assets
}

// LoadFromPath loads the pkl module at the given path and evaluates it into a Web
func LoadFromPath(ctx context.Context, path string) (ret Web, err error) {
	evaluator, err := pkl.NewEvaluator(ctx, pkl.PreconfiguredOptions)
	if err != nil {
		return ret, err
	}
	defer func() {
		cerr := evaluator.Close()
		if err == nil {
			err = cerr
		}
	}()
	ret, err = Load(ctx, evaluator, pkl.FileSource(path))
	return ret, err
}

// Load loads the pkl module at the given source and evaluates it with the given evaluator into a Web
func Load(ctx context.Context, evaluator pkl.Evaluator, source *pkl.ModuleSource) (Web, error) {
	var ret WebImpl
	err := evaluator.EvaluateModule(ctx, source, &ret)
	return ret, err
}
