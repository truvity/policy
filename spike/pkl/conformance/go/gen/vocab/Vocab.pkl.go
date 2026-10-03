// Code generated from Pkl module `spike.vocab.Vocab`. DO NOT EDIT.
package vocab

import (
	"context"

	"github.com/apple/pkl-go/pkl"
)

// The constrained vocabulary of a data contract: named types that mean the
// same thing wherever they appear, and the annotations a generator reads.
//
// `pkl:reflect` does not expose constraint lambdas, so a constraint is
// written TWICE in kind and ONCE in value:
//
//   - the alias carries the constraint as a lambda (what Pkl itself enforces);
//   - an annotation on the alias carries the same constraint as data (what a
//     generator reads).
//
// Where the value is a `local const` (every regular expression, the port
// bounds) both spellings reference it, so it cannot drift. Small literals
// (`>= 1`) are written twice, and so is the KIND (an alias that annotates
// `Range` but constrains a pattern). What covers both is the probe fixtures
// (`gen/Probes.pkl`): generated from the annotations, run against Pkl's own
// enforcement as the oracle and against every generated validator. They
// caught a real drift on their first run: see `Pattern` in Annotations.pkl.
type Vocab struct {
}

// LoadFromPath loads the pkl module at the given path and evaluates it into a Vocab
func LoadFromPath(ctx context.Context, path string) (ret Vocab, err error) {
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

// Load loads the pkl module at the given source and evaluates it with the given evaluator into a Vocab
func Load(ctx context.Context, evaluator pkl.Evaluator, source *pkl.ModuleSource) (Vocab, error) {
	var ret Vocab
	err := evaluator.EvaluateModule(ctx, source, &ret)
	return ret, err
}
