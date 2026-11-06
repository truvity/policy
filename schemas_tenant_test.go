package policy_test

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/truvity/policy"
	"github.com/truvity/policy/config"
)

// tenantSections maps each file of a tenant bundle to the schema that holds
// its shape. The manifest is the only file every bundle has.
var tenantSections = map[string]string{
	"tenant.yaml":   "manifest.json",
	"projects.yaml": "projects.json",
	"clients.yaml":  "clients.json",
	"access.yaml":   "access.json",
	"alerts.yaml":   "alerts.json",
	"status.yaml":   "status.json",
	"ci.yaml":       "ci.json",
	"registry.yaml": "registry.json",
	"github.yaml":   "github.json",
}

func tenantSchema(t *testing.T, name string) []byte {
	t.Helper()

	b, err := policy.Schemas.ReadFile(path.Join("schemas/tenant.v1", name))
	if err != nil {
		t.Fatal(err)
	}

	return b
}

func tenantDoc(t *testing.T, src string) any {
	t.Helper()

	var doc any
	if err := yaml.Unmarshal([]byte(src), &doc); err != nil {
		t.Fatal(err)
	}

	return doc
}

// Every section of every valid fixture satisfies its schema, and the full
// fixture exercises every section the contract lists. A fixture that skipped a
// section would let that section's schema drift unseen.
func TestTenantValidFixturesSatisfyTheirSchemas(t *testing.T) {
	bundles, err := filepath.Glob("tenant/testdata/valid/*")
	if err != nil || len(bundles) == 0 {
		t.Fatalf("no valid tenant fixtures found (%v): a sweep over nothing proves nothing", err)
	}

	covered := map[string]bool{}

	for _, dir := range bundles {
		files, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}

		for _, f := range files {
			schema, ok := tenantSections[f.Name()]
			if !ok {
				t.Errorf("%s: %s is not a section the contract lists", dir, f.Name())
				continue
			}

			covered[f.Name()] = true

			t.Run(path.Join(filepath.Base(dir), f.Name()), func(t *testing.T) {
				data, err := os.ReadFile(filepath.Join(dir, f.Name()))
				if err != nil {
					t.Fatal(err)
				}

				var doc any
				if err := yaml.Unmarshal(data, &doc); err != nil {
					t.Fatal(err)
				}

				if err := config.Validate(doc, tenantSchema(t, schema)); err != nil {
					t.Fatal(err)
				}
			})
		}
	}

	for name := range tenantSections {
		if !covered[name] {
			t.Errorf("no valid fixture has %s", name)
		}
	}
}

// Every section schema is strict: a field it does not name is refused. The
// one exception is the source-hosting section, whose row shape belongs to the
// tool that applies it, and which is checked for its organisation key only.
func TestTenantSectionsRefuseAnUnknownField(t *testing.T) {
	for file, schema := range tenantSections {
		if file == "github.yaml" {
			continue
		}

		t.Run(file, func(t *testing.T) {
			doc := tenantDoc(t, "schema: tenant.v1\ncompany: acme\ninterface: 1\nsurprise: 1\n")
			if file != "tenant.yaml" {
				doc = tenantDoc(t, "surprise: []\n")
			}

			if err := config.Validate(doc, tenantSchema(t, schema)); err == nil {
				t.Fatal("an unknown top-level field was accepted")
			}
		})
	}
}

func TestTenantGitHubSectionNeedsOnlyItsOrganization(t *testing.T) {
	schema := tenantSchema(t, "github.json")

	if err := config.Validate(tenantDoc(t, "organization: acme-org\nanything: [goes]\n"), schema); err != nil {
		t.Errorf("extra keys must pass: %v", err)
	}

	if err := config.Validate(tenantDoc(t, "repositories: []\n"), schema); err == nil {
		t.Error("a section with no organization was accepted")
	}
}

// A name is one name, never a pattern; a redirect never has a wildcard.
func TestTenantNamesAreNeverPatterns(t *testing.T) {
	cases := []struct {
		schema, doc string
	}{
		{"projects.json", "projects: [{name: 'shop*'}]"},
		{"projects.json", "projects: [{name: shop, endpoints: [{name: web, host: '*.acme.example'}]}]"},
		{"projects.json", "projects: [{name: Shop}]"},
		{"access.json", "grants: [{group: 'team-?', project: shop, scope: read}]"},
		{"access.json", "groups: [{name: team, members: ['*@acme.example']}]"},
		{"access.json", "grants: [{group: team, project: shop, scope: root}]"},
		{"clients.json", "clients: [{name: c, project: shop, kind: web, redirect_uris: ['https://*.acme.example/cb']}]"},
		{"clients.json", "clients: [{name: c, project: shop, kind: web, redirect_uris: ['http://acme.example/cb']}]"},
		{"ci.json", "requests: [{name: b, project: shop, repository: 'acme-org/*'}]"},
		{"manifest.json", "{schema: tenant.v1, company: Acme, interface: 1}"},
		{"manifest.json", "{schema: tenant.v1, company: acme, interface: 0}"},
		{"manifest.json", "{schema: tenant.v2, company: acme, interface: 1}"},
		{"alerts.json", "receivers: [{name: r, kind: sms, target: x}]"},
		// A webhook names a secret and never holds the address; an e-mail target
		// is one address, never a pattern.
		{"alerts.json", "receivers: [{name: r, kind: webhook, target: 'https://hooks.acme.example/T0K3N'}]"},
		{"alerts.json", "receivers: [{name: r, kind: webhook, target: 'has space'}]"},
		{"alerts.json", "receivers: [{name: r, kind: email, target: '*@acme.example'}]"},
		{"alerts.json", "receivers: [{name: r, kind: email, target: 'a b@acme.example'}]"},
		{"alerts.json", "receivers: [{name: r, kind: email, target: 'a-secret-name'}]"},
	}

	for _, c := range cases {
		t.Run(c.schema+"/"+strings.ReplaceAll(c.doc, "/", "_"), func(t *testing.T) {
			if err := config.Validate(tenantDoc(t, c.doc), tenantSchema(t, c.schema)); err == nil {
				t.Fatal("accepted")
			}
		})
	}
}

// The section that carries `why` in the contract leaves its enforcement to
// the validator, so that a missing reason has its own code. The schema must
// therefore ACCEPT a grant with no why; if it started to require one, the
// code would silently change to a generic shape failure.
func TestTenantGrantWithoutWhyIsTheValidatorsToRefuse(t *testing.T) {
	doc := tenantDoc(t, "grants: [{group: team, project: shop, scope: read}]")
	if err := config.Validate(doc, tenantSchema(t, "access.json")); err != nil {
		t.Fatalf("the schema must leave a missing why to the validator: %v", err)
	}
}
