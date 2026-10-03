// Code generated from Pkl module `spike.contract.urlshortener.Shortener`. DO NOT EDIT.
package shortener

import (
	"context"

	"github.com/apple/pkl-go/pkl"
	"spike.invalid/pklconf/gen/fragments"
	"spike.invalid/pklconf/gen/service"
)

type Shortener interface {
	service.Service

	GetListen() fragments.Listen

	GetBaseURL() string

	GetDatabase() fragments.Postgres
}

var _ Shortener = ShortenerImpl{}

type ShortenerImpl struct {
	service.ServiceImpl

	Listen fragments.Listen `pkl:"listen"`

	BaseURL string `pkl:"baseURL"`

	Database fragments.Postgres `pkl:"database"`
}

func (rcv ShortenerImpl) GetListen() fragments.Listen {
	return rcv.Listen
}

func (rcv ShortenerImpl) GetBaseURL() string {
	return rcv.BaseURL
}

func (rcv ShortenerImpl) GetDatabase() fragments.Postgres {
	return rcv.Database
}

// LoadFromPath loads the pkl module at the given path and evaluates it into a Shortener
func LoadFromPath(ctx context.Context, path string) (ret Shortener, err error) {
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

// Load loads the pkl module at the given source and evaluates it with the given evaluator into a Shortener
func Load(ctx context.Context, evaluator pkl.Evaluator, source *pkl.ModuleSource) (Shortener, error) {
	var ret ShortenerImpl
	err := evaluator.EvaluateModule(ctx, source, &ret)
	return ret, err
}
