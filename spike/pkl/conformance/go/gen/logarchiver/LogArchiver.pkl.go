// Code generated from Pkl module `spike.contract.urlshortener.LogArchiver`. DO NOT EDIT.
package logarchiver

import (
	"context"

	"github.com/apple/pkl-go/pkl"
	"spike.invalid/pklconf/gen/service"
)

type LogArchiver interface {
	service.Service

	GetEvents() Events

	GetArchive() Archive
}

var _ LogArchiver = LogArchiverImpl{}

// The archiver: consume what the redirect service recorded about each request, and write it to an object store as NDJSON. It owns no database and answers no calls.
type LogArchiverImpl struct {
	service.ServiceImpl

	Events Events `pkl:"events"`

	Archive Archive `pkl:"archive"`
}

func (rcv LogArchiverImpl) GetEvents() Events {
	return rcv.Events
}

func (rcv LogArchiverImpl) GetArchive() Archive {
	return rcv.Archive
}

// LoadFromPath loads the pkl module at the given path and evaluates it into a LogArchiver
func LoadFromPath(ctx context.Context, path string) (ret LogArchiver, err error) {
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

// Load loads the pkl module at the given source and evaluates it with the given evaluator into a LogArchiver
func Load(ctx context.Context, evaluator pkl.Evaluator, source *pkl.ModuleSource) (LogArchiver, error) {
	var ret LogArchiverImpl
	err := evaluator.EvaluateModule(ctx, source, &ret)
	return ret, err
}
