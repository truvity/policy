// Code generated from Pkl module `spike.contract.Fragments`. DO NOT EDIT.
package fragments

import (
	"context"

	"github.com/apple/pkl-go/pkl"
)

// The shapes that mean the same thing in more than one service (schemas/fragments/).
type Fragments struct {
}

// LoadFromPath loads the pkl module at the given path and evaluates it into a Fragments
func LoadFromPath(ctx context.Context, path string) (ret Fragments, err error) {
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

// Load loads the pkl module at the given source and evaluates it with the given evaluator into a Fragments
func Load(ctx context.Context, evaluator pkl.Evaluator, source *pkl.ModuleSource) (Fragments, error) {
	var ret Fragments
	err := evaluator.EvaluateModule(ctx, source, &ret)
	return ret, err
}
