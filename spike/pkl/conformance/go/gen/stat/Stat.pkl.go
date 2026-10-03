// Code generated from Pkl module `spike.contract.urlshortener.Stat`. DO NOT EDIT.
package stat

import (
	"context"

	"github.com/apple/pkl-go/pkl"
	"spike.invalid/pklconf/gen/fragments"
	"spike.invalid/pklconf/gen/service"
)

type Stat interface {
	service.Service

	GetUrls() Urls

	GetEvents() Events

	GetTls() *fragments.TlsFields
}

var _ Stat = StatImpl{}

// The click counter: consume redirects, and ask the service that owns the table to count them.
type StatImpl struct {
	service.ServiceImpl

	// The service that owns the URL tables. An address and nothing else: which protocol the caller speaks is in its code, and whether the connection is authenticated is the tls block's business.
	Urls Urls `pkl:"urls"`

	Events Events `pkl:"events"`

	Tls *fragments.TlsFields `pkl:"tls"`
}

// The service that owns the URL tables. An address and nothing else: which protocol the caller speaks is in its code, and whether the connection is authenticated is the tls block's business.
func (rcv StatImpl) GetUrls() Urls {
	return rcv.Urls
}

func (rcv StatImpl) GetEvents() Events {
	return rcv.Events
}

func (rcv StatImpl) GetTls() *fragments.TlsFields {
	return rcv.Tls
}

// LoadFromPath loads the pkl module at the given path and evaluates it into a Stat
func LoadFromPath(ctx context.Context, path string) (ret Stat, err error) {
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

// Load loads the pkl module at the given source and evaluates it with the given evaluator into a Stat
func Load(ctx context.Context, evaluator pkl.Evaluator, source *pkl.ModuleSource) (Stat, error) {
	var ret StatImpl
	err := evaluator.EvaluateModule(ctx, source, &ret)
	return ret, err
}
