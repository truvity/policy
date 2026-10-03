// Code generated from Pkl module `spike.contract.Service`. DO NOT EDIT.
package service

import (
	"context"

	"github.com/apple/pkl-go/pkl"
	"spike.invalid/pklconf/gen/fragments"
)

type Service interface {
	GetProbes() fragments.Probes

	GetLog() *fragments.Log

	GetDrain() *fragments.Drain
}

var _ Service = ServiceImpl{}

// The envelope every service's configuration carries. A service references this from its own schema with allOf, and adds its own properties beside it. What is in here is what EVERY component has, including a job that exits and a consumer that answers nothing: somewhere to report health, a log level, and a shutdown budget. A listener is not one of those — see docs/contracts/config.md.
type ServiceImpl struct {
	Probes fragments.Probes `pkl:"probes"`

	Log *fragments.Log `pkl:"log"`

	Drain *fragments.Drain `pkl:"drain"`
}

func (rcv ServiceImpl) GetProbes() fragments.Probes {
	return rcv.Probes
}

func (rcv ServiceImpl) GetLog() *fragments.Log {
	return rcv.Log
}

func (rcv ServiceImpl) GetDrain() *fragments.Drain {
	return rcv.Drain
}

// LoadFromPath loads the pkl module at the given path and evaluates it into a Service
func LoadFromPath(ctx context.Context, path string) (ret Service, err error) {
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

// Load loads the pkl module at the given source and evaluates it with the given evaluator into a Service
func Load(ctx context.Context, evaluator pkl.Evaluator, source *pkl.ModuleSource) (Service, error) {
	var ret ServiceImpl
	err := evaluator.EvaluateModule(ctx, source, &ret)
	return ret, err
}
