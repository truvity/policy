package tenant_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/truvity/policy/tenant"
)

// registration is the platform's record of the company the fixtures are for.
func registration() tenant.Registration {
	return tenant.Registration{
		Slug:         "acme",
		GitHubOrg:    "acme-org",
		GitHubOrgID:  1,
		EmailDomains: []string{"acme.example"},
		Zones:        []string{"acme.example"},
		Registries:   []string{"shared"},
		Ceilings: map[string]json.RawMessage{
			tenant.AccessDomain: json.RawMessage(`{"max_scope":"write"}`),
		},
	}
}

// estate is a platform that reserves one group name, knows one other
// company's project and zone, and bounds access by a maximum scope. It is the
// shape a caller implements; nothing here is the checker's own.
type estate struct{}

func (estate) ProjectOwner(project string) (string, bool) {
	if project == "inventory" {
		return "globex", true
	}

	return "", false
}

func (estate) ProviderOwned(kind, name string) bool {
	return kind == tenant.KindGroup && name == "platform-admins"
}

func (estate) ZoneOwner(zone string) (string, bool) {
	switch zone {
	case "acme.example":
		return "acme", true
	case "globex.example":
		return "globex", true
	}

	return "", false
}

var scopeRank = map[string]int{"read": 1, "write": 2, "admin": 3}

func (estate) WithinCeiling(domain string, ceiling json.RawMessage, row any) error {
	if domain != tenant.AccessDomain {
		return fmt.Errorf("no ceiling for %q", domain)
	}

	var c struct {
		MaxScope string `json:"max_scope"`
	}

	if err := json.Unmarshal(ceiling, &c); err != nil {
		return err
	}

	g, ok := row.(tenant.Grant)
	if !ok {
		return fmt.Errorf("not a grant: %T", row)
	}

	if scopeRank[g.Scope] > scopeRank[c.MaxScope] {
		return fmt.Errorf("scope %q exceeds ceiling %q", g.Scope, c.MaxScope)
	}

	return nil
}

const (
	validFull         = "testdata/valid/full"
	validManifestOnly = "testdata/valid/manifest-only"
)

// bundle is the valid full bundle with the files of overlay dirs (and then of
// edits) laid over it: a refusal fixture is a valid bundle with one change, so
// it holds only the file it changes.
func bundle(t *testing.T, overlay string, edits map[string]string) fs.FS {
	t.Helper()

	m := fstest.MapFS{}

	for _, dir := range []string{validFull, overlay} {
		if dir == "" {
			continue
		}

		files, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}

		for _, f := range files {
			b, err := os.ReadFile(filepath.Join(dir, f.Name()))
			if err != nil {
				t.Fatal(err)
			}

			m[f.Name()] = &fstest.MapFile{Data: b}
		}
	}

	for name, content := range edits {
		if content == "" {
			delete(m, name)
			continue
		}

		m[name] = &fstest.MapFile{Data: []byte(content)}
	}

	return m
}

func mustLoad(t *testing.T, fsys fs.FS) *tenant.Bundle {
	t.Helper()

	b, err := tenant.Load(fsys)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	return b
}

func asErrors(t *testing.T, err error) tenant.Errors {
	t.Helper()

	var es tenant.Errors
	if !errors.As(err, &es) {
		t.Fatalf("want tenant.Errors, got %T: %v", err, err)
	}

	return es
}

// A refusal fixture is a valid bundle with one change. Each names the level
// that must refuse it (a refusal that needs no platform must not wait for
// one), the code, and the exact place.
var refusals = map[tenant.Code]struct {
	level      string // "load", "validate" or "check"
	file, path string
}{
	tenant.UnknownField:     {"load", "projects.yaml", "projects[0].colour"},
	tenant.Invalid:          {"load", "projects.yaml", "projects[0].name"},
	tenant.UnknownInterface: {"load", "tenant.yaml", "interface"},
	tenant.Duplicate:        {"validate", "projects.yaml", "projects[2].name"},
	tenant.MissingReason:    {"validate", "access.yaml", "grants[1].why"},
	tenant.CrossCompanyRef:  {"validate", "access.yaml", "grants[0].project"},
	tenant.ProviderOwned:    {"check", "access.yaml", "groups[0].name"},
	tenant.CeilingExceeded:  {"check", "access.yaml", "grants[2]"},
	tenant.WrongOrg:         {"check", "github.yaml", "organization"},
}

func TestEveryRefusalFixtureIsRefusedWithItsCodeAtItsPath(t *testing.T) {
	for _, code := range tenant.Codes() {
		want, ok := refusals[code]
		if !ok {
			t.Errorf("code %q has no entry in the refusal table", code)
			continue
		}

		t.Run(string(code), func(t *testing.T) {
			fsys := bundle(t, path.Join("testdata/refuse", string(code)), nil)

			var err error

			switch want.level {
			case "load":
				_, err = tenant.Load(fsys)
			case "validate":
				err = mustLoad(t, fsys).Validate()
			case "check":
				b := mustLoad(t, fsys)
				if err := b.Validate(); err != nil {
					t.Fatalf("the fixture must pass Validate and fail only at Check: %v", err)
				}

				err = b.Check(registration(), estate{})
			default:
				t.Fatalf("bad level %q", want.level)
			}

			if err == nil {
				t.Fatalf("%s: the bundle was accepted", code)
			}

			es := asErrors(t, err)
			if len(es) != 1 {
				t.Fatalf("a fixture is a valid bundle with ONE change; got %d errors:\n%v", len(es), err)
			}

			got := es[0]
			if got.Code != code || got.File != want.file || got.Path != want.path {
				t.Errorf("got %s %s %s, want %s %s %s\n%v", got.Code, got.File, got.Path, code, want.file, want.path, err)
			}

			if got.Reason == "" {
				t.Error("a refusal with no reason says nothing")
			}

			if !strings.Contains(got.Error(), "["+string(code)+"]") || !strings.HasPrefix(got.Error(), "tenant ") {
				t.Errorf("the message must end in its code: %q", got.Error())
			}
		})
	}
}

// A guard that sweeps nothing proves nothing: every code needs a fixture
// directory with a file in it, and every directory must be a code, so a
// fixture can neither be forgotten nor orphaned.
func TestEveryCodeHasAFixtureAndEveryFixtureIsACode(t *testing.T) {
	dirs, err := os.ReadDir("testdata/refuse")
	if err != nil || len(dirs) == 0 {
		t.Fatalf("no refusal fixtures found (%v)", err)
	}

	isCode := map[string]bool{}
	for _, c := range tenant.Codes() {
		isCode[string(c)] = true
	}

	have := map[string]bool{}

	for _, d := range dirs {
		have[d.Name()] = true

		if !isCode[d.Name()] {
			t.Errorf("testdata/refuse/%s is not a code", d.Name())
		}

		files, err := os.ReadDir(filepath.Join("testdata/refuse", d.Name()))
		if err != nil || len(files) == 0 {
			t.Errorf("testdata/refuse/%s is empty", d.Name())
		}
	}

	for c := range isCode {
		if !have[c] {
			t.Errorf("code %q has no fixture directory", c)
		}
	}
}

func TestValidFullBundleIsAcceptedAtEveryLevel(t *testing.T) {
	b := mustLoad(t, bundle(t, "", nil))

	if err := b.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	if err := b.Check(registration(), estate{}); err != nil {
		t.Fatalf("Check: %v", err)
	}

	if b.Manifest != (tenant.Manifest{Schema: "tenant.v1", Company: "acme", Interface: 1}) {
		t.Errorf("manifest: %+v", b.Manifest)
	}

	if len(b.Projects) != 2 || len(b.Projects[0].Endpoints) != 2 || b.Projects[0].Endpoints[0].Host != "shop.acme.example" {
		t.Errorf("projects: %+v", b.Projects)
	}

	if len(b.Clients) != 2 || len(b.Access.Groups) != 2 || len(b.Access.Grants) != 2 ||
		len(b.Alerts.Receivers) != 2 || len(b.Alerts.Routes) != 2 || len(b.Status) != 1 ||
		len(b.CI) != 1 || len(b.Registry) != 1 {
		t.Errorf("sections were not all decoded: %+v", b)
	}

	if b.GitHub == nil || b.GitHub.Organization != "acme-org" || b.GitHub.Rest["repositories"] == nil {
		t.Errorf("github: %+v", b.GitHub)
	}
}

// A manifest-only bundle is valid and asks for nothing, and every list in it
// is empty rather than nil: an omitted section and an empty one are alike.
func TestManifestOnlyBundleAsksForNothing(t *testing.T) {
	m := fstest.MapFS{"tenant.yaml": &fstest.MapFile{Data: []byte("schema: tenant.v1\ncompany: acme\ninterface: 1\n")}}

	b := mustLoad(t, m)

	if err := b.Check(tenant.Registration{Slug: "acme"}, nil); err != nil {
		t.Fatalf("a manifest-only bundle must pass with an empty registration: %v", err)
	}

	if b.Projects == nil || b.Clients == nil || b.Access.Groups == nil || b.Access.Grants == nil ||
		b.Alerts.Receivers == nil || b.Alerts.Routes == nil || b.Status == nil || b.CI == nil || b.Registry == nil {
		t.Errorf("an omitted section must load as an empty list, not nil: %+v", b)
	}

	if b.GitHub != nil {
		t.Errorf("no github.yaml, no github rows: %+v", b.GitHub)
	}
}

func TestLoadRefusesWhatIsNotABundle(t *testing.T) {
	cases := map[string]struct {
		edits map[string]string
		code  tenant.Code
		file  string
	}{
		"no manifest":                           {map[string]string{"tenant.yaml": ""}, tenant.Invalid, "tenant.yaml"},
		"a file the contract does not name":     {map[string]string{"notes.yaml": "x: 1\n"}, tenant.UnknownField, "notes.yaml"},
		"a newer interface":                     {map[string]string{"tenant.yaml": "schema: tenant.v1\ncompany: acme\ninterface: 3\n"}, tenant.UnknownInterface, "tenant.yaml"},
		"another bundle format":                 {map[string]string{"tenant.yaml": "schema: tenant.v2\ncompany: acme\ninterface: 1\n"}, tenant.UnknownInterface, "tenant.yaml"},
		"not YAML":                              {map[string]string{"projects.yaml": "projects: [\n"}, tenant.Invalid, "projects.yaml"},
		"a section of the wrong type":           {map[string]string{"projects.yaml": "projects: shop\n"}, tenant.Invalid, "projects.yaml"},
		"a wildcard host":                       {map[string]string{"projects.yaml": "projects:\n  - name: shop\n    endpoints:\n      - {name: web, host: '*.acme.example'}\n"}, tenant.Invalid, "projects.yaml"},
		"a wildcard redirect":                   {map[string]string{"clients.yaml": "clients:\n  - {name: c, project: shop, kind: web, redirect_uris: ['https://*.acme.example/cb']}\n"}, tenant.Invalid, "clients.yaml"},
		"a github section with no organization": {map[string]string{"github.yaml": "repositories: []\n"}, tenant.Invalid, "github.yaml"},
		"a second YAML document":                {map[string]string{"projects.yaml": "projects:\n  - name: shop\n---\nprojects:\n  - name: Not_A_Name\n"}, tenant.Invalid, "projects.yaml"},
		"a webhook target that is a URL":        {map[string]string{"alerts.yaml": "receivers:\n  - {name: pager, kind: webhook, target: 'https://hooks.acme.example/T0K3N'}\n"}, tenant.Invalid, "alerts.yaml"},
		"an e-mail target that is a pattern":    {map[string]string{"alerts.yaml": "receivers:\n  - {name: oncall, kind: email, target: '*@acme.example'}\n"}, tenant.Invalid, "alerts.yaml"},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := tenant.Load(bundle(t, "", c.edits))
			if err == nil {
				t.Fatal("accepted")
			}

			es := asErrors(t, err)
			if !es.Has(c.code) {
				t.Errorf("want %s in:\n%v", c.code, err)
			}

			for _, e := range es {
				if e.File != c.file {
					t.Errorf("want errors only in %s, got %v", c.file, e)
				}
			}
		})
	}
}

// An unknown field names the field, so a person reading the message is not
// left to find it.
func TestUnknownFieldIsNamedInThePath(t *testing.T) {
	_, err := tenant.Load(bundle(t, "testdata/refuse/unknown-field", nil))

	es := asErrors(t, err)
	if len(es) != 1 || es[0].Path != "projects[0].colour" {
		t.Fatalf("%v", err)
	}
}

// A refusal reports every failure it finds, not the first: a company that has
// to fix and resubmit once per row will stop reading the message.
func TestARefusalReportsEveryFailureInOnePass(t *testing.T) {
	b := mustLoad(t, bundle(t, "", map[string]string{
		"access.yaml": "groups:\n  - name: team\ngrants:\n" +
			"  - {group: team, project: shop, scope: read}\n" + // no why
			"  - {group: team, project: ghost, scope: read, why: x}\n" + // undefined project
			"  - {group: nobody, project: shop, scope: read, why: x}\n", // undefined group
		"status.yaml": "entities:\n  - {name: a, project: shop, title: A}\n  - {name: a, project: shop, title: A}\n",
	}))

	es := asErrors(t, b.Validate())
	for _, want := range []tenant.Code{tenant.MissingReason, tenant.CrossCompanyRef, tenant.Duplicate} {
		if !es.Has(want) {
			t.Errorf("missing %s in:\n%v", want, es)
		}
	}

	if len(es) != 4 {
		t.Errorf("want 4 failures (no why, undefined project, undefined group, duplicate entity), got %d:\n%v", len(es), es)
	}
}

func TestValidateAlsoRefusesABundleBuiltInCode(t *testing.T) {
	b := &tenant.Bundle{
		Manifest: tenant.Manifest{Schema: tenant.Schema, Company: "acme", Interface: 1},
		Projects: []tenant.Project{{Name: "shop*"}},
		Access: tenant.Access{
			Groups: []tenant.Group{{Name: "team"}},
			Grants: []tenant.Grant{{Group: "team", Project: "shop*", Scope: "read", Why: "  "}},
		},
	}

	es := asErrors(t, b.Validate())
	if !es.Has(tenant.Invalid) || !es.Has(tenant.MissingReason) {
		t.Errorf("a glob in a name and a blank reason must both be refused:\n%v", es)
	}
}

// The checks that need the platform, one change each.
func TestCheckRefusesWhatTheRegistrationDoesNotAllow(t *testing.T) {
	proj := func(host string) string {
		return "projects:\n  - name: shop\n    endpoints:\n      - {name: web, host: " + host + "}\n"
	}

	cases := map[string]struct {
		edits map[string]string
		reg   func(*tenant.Registration)
		code  tenant.Code
		file  string
		path  string
	}{
		"a host outside every zone the company owns": {
			map[string]string{"projects.yaml": proj("shop.globex.example")}, nil,
			tenant.CeilingExceeded, "projects.yaml", "projects[0].endpoints[0].host"},
		"a host in a zone another company owns": {
			map[string]string{"projects.yaml": proj("shop.globex.example")},
			func(r *tenant.Registration) { r.Zones = []string{"globex.example", "acme.example"} },
			tenant.CeilingExceeded, "projects.yaml", "projects[0].endpoints[0].host"},
		"a host that only ends like a zone": {
			map[string]string{"projects.yaml": proj("evilacme.example")}, nil,
			tenant.CeilingExceeded, "projects.yaml", "projects[0].endpoints[0].host"},
		"a member outside the company's e-mail domains": {
			map[string]string{"access.yaml": "groups:\n  - {name: team, members: [mallory@globex.example]}\n"}, nil,
			tenant.CeilingExceeded, "access.yaml", "groups[0].members[0]"},
		"a registry the company does not hold": {
			map[string]string{"registry.yaml": "projects:\n  - {project: shop, registry: other}\n"}, nil,
			tenant.CeilingExceeded, "registry.yaml", "projects[0].registry"},
		"a project name another company holds": {
			map[string]string{"projects.yaml": "projects:\n  - name: inventory\n"}, nil,
			tenant.Duplicate, "projects.yaml", "projects[0].name"},
		"a CI request for another organisation's repository": {
			map[string]string{"ci.yaml": "requests:\n  - {name: b, project: shop, repository: globex-org/shop}\n"}, nil,
			tenant.WrongOrg, "ci.yaml", "requests[0].repository"},
		"a manifest for another company": {
			map[string]string{"tenant.yaml": "schema: tenant.v1\ncompany: globex\ninterface: 1\n"}, nil,
			tenant.CrossCompanyRef, "tenant.yaml", "company"},
		"a grant when the company has no access ceiling": {
			nil, func(r *tenant.Registration) { r.Ceilings = nil },
			tenant.CeilingExceeded, "access.yaml", "grants[0]"},
		"a source-hosting row when none is registered": {
			nil, func(r *tenant.Registration) { r.GitHubOrg = "" },
			tenant.WrongOrg, "github.yaml", "organization"},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			reg := registration()
			if c.reg != nil {
				c.reg(&reg)
			}

			b := mustLoad(t, bundle(t, "", c.edits))

			es := asErrors(t, b.Check(reg, estate{}))
			found := false

			for _, e := range es {
				if e.Code == c.code && e.File == c.file && e.Path == c.path {
					found = true
				}
			}

			if !found {
				t.Errorf("want %s %s %s in:\n%v", c.code, c.file, c.path, es)
			}
		})
	}
}

// An omitted ceiling admits nothing, and so does an absent platform.
func TestNoPlatformAdmitsNothingThatNeedsOne(t *testing.T) {
	b := mustLoad(t, bundle(t, "", nil))

	es := asErrors(t, b.Check(registration(), nil))
	if !es.Has(tenant.CeilingExceeded) {
		t.Errorf("a nil estate must refuse grants, whose ceiling it cannot read:\n%v", es)
	}
}

// Check refuses the same set at every call: the order is the file order and
// then the row order, so a caller can diff two runs.
func TestErrorsAreDeterministic(t *testing.T) {
	b := mustLoad(t, bundle(t, "testdata/refuse/ceiling-exceeded", map[string]string{"github.yaml": "organization: globex-org\n"}))

	first := b.Check(registration(), estate{}).Error()
	for range 20 {
		if got := b.Check(registration(), estate{}).Error(); got != first {
			t.Fatalf("order changed:\n%s\n---\n%s", first, got)
		}
	}
}

// Only the first document of a file is read, so a second one would be content
// no check ever saw and the pinned bundle still carries.
func TestLoadRefusesASecondDocumentButNotAnEmptyOne(t *testing.T) {
	_, err := tenant.Load(bundle(t, "", map[string]string{"projects.yaml": "projects:\n  - name: shop\n---\nprojects:\n  - name: other\n"}))
	if err == nil || !asErrors(t, err).Has(tenant.Invalid) {
		t.Fatalf("a second document was accepted: %v", err)
	}

	if _, err := tenant.Load(bundle(t, "", map[string]string{"projects.yaml": "projects:\n  - name: shop\n---\n"})); err != nil {
		t.Fatalf("an empty trailing document holds nothing and is harmless: %v", err)
	}
}

// Digest refuses a link; so must Load, or a link outside the bundle is read
// by Load and refused only when the bundle is pinned.
func TestLoadRefusesAFileThatIsNotRegular(t *testing.T) {
	m := bundle(t, "", nil).(fstest.MapFS)
	m["projects.yaml"] = &fstest.MapFile{Data: []byte("../elsewhere/projects.yaml"), Mode: fs.ModeSymlink | 0o777}

	_, err := tenant.Load(m)
	if err == nil {
		t.Fatal("a linked section was accepted")
	}

	es := asErrors(t, err)
	if len(es) != 1 || es[0].Code != tenant.Invalid || es[0].File != "projects.yaml" {
		t.Errorf("want one invalid refusal of projects.yaml:\n%v", err)
	}
}

// A webhook target names a secret and an e-mail target is one address. The
// schema says so for files; Validate says so for a bundle built in code.
func TestValidateRefusesATargetThatHoldsASecretOrAPattern(t *testing.T) {
	for name, r := range map[string]tenant.Receiver{
		"a webhook URL":          {Name: "pager", Kind: "webhook", Target: "https://hooks.acme.example/T0K3N"},
		"a webhook with a space": {Name: "pager", Kind: "webhook", Target: "has space"},
		"an e-mail pattern":      {Name: "oncall", Kind: "email", Target: "*@acme.example"},
		"an e-mail with a space": {Name: "oncall", Kind: "email", Target: "a b@acme.example"},
	} {
		t.Run(name, func(t *testing.T) {
			b := &tenant.Bundle{
				Manifest: tenant.Manifest{Schema: tenant.Schema, Company: "acme", Interface: 1},
				Alerts:   tenant.Alerts{Receivers: []tenant.Receiver{r}},
			}

			es := asErrors(t, b.Validate())
			if len(es) != 1 || es[0].Code != tenant.Invalid || es[0].Path != "receivers[0].target" {
				t.Errorf("want one invalid refusal at receivers[0].target:\n%v", es)
			}
		})
	}
}

// Two endpoints on one host make an ambiguous route, in one project or in two.
func TestValidateRefusesAHostClaimedTwice(t *testing.T) {
	b := mustLoad(t, bundle(t, "", map[string]string{"projects.yaml": "projects:\n" +
		"  - name: shop\n    endpoints:\n      - {name: web, host: x.acme.example}\n      - {name: api, host: x.acme.example}\n" +
		"  - name: ledger\n    endpoints:\n      - {name: web, host: x.acme.example}\n",
		"clients.yaml": "", "access.yaml": "", "alerts.yaml": "", "status.yaml": "", "ci.yaml": "", "registry.yaml": ""}))

	es := asErrors(t, b.Validate())
	want := []string{"projects[0].endpoints[1].host", "projects[1].endpoints[0].host"}

	if len(es) != len(want) {
		t.Fatalf("want %d refusals:\n%v", len(want), es)
	}

	for i, e := range es {
		if e.Code != tenant.Duplicate || e.Path != want[i] {
			t.Errorf("got %s %s, want duplicate %s", e.Code, e.Path, want[i])
		}
	}
}

// Hosts are lower case without a trailing dot; a registration written another
// way must not make every host fail.
func TestCheckReadsZonesCaseAndDotInsensitively(t *testing.T) {
	b := mustLoad(t, bundle(t, "", nil))

	reg := registration()
	reg.Zones = []string{"Acme.Example."}

	if err := b.Check(reg, estate{}); err != nil {
		t.Fatalf("a zone spelled with capitals and a trailing dot refused every host: %v", err)
	}
}
