// Code generated from Pkl module `spike.contract.urlshortener.Redirect`. DO NOT EDIT.
package redirect

import (
	"context"

	"github.com/apple/pkl-go/pkl"
	"spike.invalid/pklconf/gen/fragments"
	"spike.invalid/pklconf/gen/service"
)

type Redirect interface {
	service.Service

	GetListen() fragments.Listen

	GetTls() *fragments.TlsFields

	GetEvents() Events
}

var _ Redirect = RedirectImpl{}

// The redirect service: resolve a short key and emit what happened.
type RedirectImpl struct {
	service.ServiceImpl

	Listen fragments.Listen `pkl:"listen"`

	Tls *fragments.TlsFields `pkl:"tls"`

	Events Events `pkl:"events"`
}

func (rcv RedirectImpl) GetListen() fragments.Listen {
	return rcv.Listen
}

func (rcv RedirectImpl) GetTls() *fragments.TlsFields {
	return rcv.Tls
}

func (rcv RedirectImpl) GetEvents() Events {
	return rcv.Events
}

// LoadFromPath loads the pkl module at the given path and evaluates it into a Redirect
func LoadFromPath(ctx context.Context, path string) (ret Redirect, err error) {
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

// Load loads the pkl module at the given source and evaluates it with the given evaluator into a Redirect
func Load(ctx context.Context, evaluator pkl.Evaluator, source *pkl.ModuleSource) (Redirect, error) {
	var ret RedirectImpl
	err := evaluator.EvaluateModule(ctx, source, &ret)
	return ret, err
}
