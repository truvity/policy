// Code generated from Pkl module `spike.contract.urlshortener.Urls`. DO NOT EDIT.
package urls

import (
	"context"

	"github.com/apple/pkl-go/pkl"
	"spike.invalid/pklconf/gen/fragments"
	"spike.invalid/pklconf/gen/service"
)

type Urls interface {
	service.Service

	GetListen() fragments.Listen

	GetTls() *fragments.TlsFields
}

var _ Urls = UrlsImpl{}

// The service that owns the URL tables. Everything that writes them asks it.
type UrlsImpl struct {
	service.ServiceImpl

	Listen fragments.Listen `pkl:"listen"`

	Tls *fragments.TlsFields `pkl:"tls"`
}

func (rcv UrlsImpl) GetListen() fragments.Listen {
	return rcv.Listen
}

func (rcv UrlsImpl) GetTls() *fragments.TlsFields {
	return rcv.Tls
}

// LoadFromPath loads the pkl module at the given path and evaluates it into a Urls
func LoadFromPath(ctx context.Context, path string) (ret Urls, err error) {
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

// Load loads the pkl module at the given source and evaluates it with the given evaluator into a Urls
func Load(ctx context.Context, evaluator pkl.Evaluator, source *pkl.ModuleSource) (Urls, error) {
	var ret UrlsImpl
	err := evaluator.EvaluateModule(ctx, source, &ret)
	return ret, err
}
