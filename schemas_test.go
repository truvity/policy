package policy_test

import (
	"io/fs"
	"path"
	"strings"
	"testing"

	"github.com/truvity/policy"
	"github.com/truvity/policy/config"
)

// Every published schema compiles, and every `$id` matches the path it is
// stored at.
//
// The second half is what stops a subtle break: the loader registers each
// embedded document under an identifier derived from its PATH, so a document
// whose own `$id` disagrees is registered under a name nothing refers to.
// Every `$ref` to it then fails to resolve — or worse, resolves to a stale
// copy — and the only symptom is a schema that validates less than it says.
func TestEveryPublishedSchemaCompilesAndIsIdentifiedByItsPath(t *testing.T) {
	err := fs.WalkDir(policy.Schemas, "schemas", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || path.Ext(p) != ".json" {
			return err
		}
		t.Run(p, func(t *testing.T) {
			b, err := policy.Schemas.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}

			// Compiling it as a service's own schema exercises the same path
			// the loader takes, references included.
			if err := config.Validate(map[string]any{}, b); err != nil {
				var ce *config.Error
				if !asConfigError(err, &ce) || ce.Err != nil {
					t.Fatalf("does not compile: %v", err)
				}
				// Refusing an empty document is a valid answer; failing to
				// compile is not.
			}

			want := policy.SchemaBase + strings.TrimPrefix(p, "schemas/")
			if got := idOf(t, b); got != want {
				t.Errorf("$id is %q, want %q — a document registered under a name nothing refers to is a reference that silently resolves to nothing", got, want)
			}
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func idOf(t *testing.T, schema []byte) string {
	t.Helper()
	var m map[string]any
	if err := jsonUnmarshal(schema, &m); err != nil {
		t.Fatal(err)
	}
	id, _ := m["$id"].(string)
	return id
}
