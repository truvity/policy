// Code generated from Pkl module `spike.contract.urlshortener.Migrate`. DO NOT EDIT.
package migrate

import (
	"context"

	"github.com/apple/pkl-go/pkl"
	"spike.invalid/pklconf/gen/fragments"
)

// The migration job. It does NOT reference the service envelope: a job that runs once and exits is not a service, and probes it never serves would be configuration a deployment can set and watch do nothing.
type Migrate struct {
	Log *fragments.Log `pkl:"log"`

	// The role the migration becomes before it creates anything, so that tables are owned by the owner rather than by whoever migrated. Removing the migration user must not orphan the schema.
	OwnerRole string `pkl:"ownerRole"`

	// The role the running components use, granted on the app schemas after the migration lands. Unset skips the grant, which is correct where the platform grants it instead.
	AppRole *string `pkl:"appRole"`
}

// LoadFromPath loads the pkl module at the given path and evaluates it into a Migrate
func LoadFromPath(ctx context.Context, path string) (ret Migrate, err error) {
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

// Load loads the pkl module at the given source and evaluates it with the given evaluator into a Migrate
func Load(ctx context.Context, evaluator pkl.Evaluator, source *pkl.ModuleSource) (Migrate, error) {
	var ret Migrate
	err := evaluator.EvaluateModule(ctx, source, &ret)
	return ret, err
}
