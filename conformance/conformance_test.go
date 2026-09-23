package conformance_test

import (
	"os"
	"testing"

	"github.com/truvity/policy/conformance"
)

type config struct {
	Probes struct {
		Address string `json:"address"`
	} `json:"probes"`
	BaseURL  string `json:"baseURL"`
	Database struct {
		URL            string `json:"url"`
		PasswordEnv    string `json:"passwordEnv"`
		MaxConnections int    `json:"maxConnections"`
	} `json:"database"`
}

func schema(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/drift.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// recorder captures what the helper would report, so that the helpers can be
// tested for failing when they should. A helper that never fails is a helper
// nobody should trust, and the only way to know is to make it fail.
type recorder struct {
	testing.TB
	errs []string
}

func (r *recorder) Errorf(format string, args ...any) { r.errs = append(r.errs, format) }
func (r *recorder) Helper()                           {}

func TestTypeMatchesSchemaAcceptsAgreement(t *testing.T) {
	r := &recorder{TB: t}
	conformance.TypeMatchesSchema(r, config{}, schema(t))
	if len(r.errs) != 0 {
		t.Errorf("a type and a schema that agree were reported as drift: %v", r.errs)
	}
}

func TestTypeMatchesSchemaCatchesAFieldTheSchemaLacks(t *testing.T) {
	type withExtra struct {
		config
		Extra string `json:"extra"`
	}
	r := &recorder{TB: t}
	conformance.TypeMatchesSchema(r, withExtra{}, schema(t))
	if len(r.errs) == 0 {
		t.Error("a field the schema does not describe was not reported; nothing would stop a deployment omitting it")
	}
}

func TestTypeMatchesSchemaCatchesARenamedField(t *testing.T) {
	// The failure this exists for: renamed on one side only.
	type renamed struct {
		Probes struct {
			Address string `json:"address"`
		} `json:"probes"`
		BaseURL  string `json:"base_url"` // was baseURL
		Database struct {
			URL            string `json:"url"`
			PasswordEnv    string `json:"passwordEnv"`
			MaxConnections int    `json:"maxConnections"`
		} `json:"database"`
	}
	r := &recorder{TB: t}
	conformance.TypeMatchesSchema(r, renamed{}, schema(t))
	if len(r.errs) < 2 {
		t.Errorf("a rename should be reported from both sides, got %d: %v", len(r.errs), r.errs)
	}
}

func TestConfigMapDataFindsTheKey(t *testing.T) {
	rendered := []byte(`apiVersion: v1
kind: Service
metadata:
  name: shortener
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: shortener-config
data:
  config.yaml: |
    probes:
      address: ":7070"
`)
	got := conformance.ConfigMapData(t, rendered, "config.yaml")
	if len(got) == 0 {
		t.Fatal("nothing returned")
	}
	conformance.ValidDocument(t, got, []byte(`{
      "$schema": "https://json-schema.org/draft/2020-12/schema",
      "$ref": "https://github.com/truvity/policy/schemas/service.json"
    }`))
}
