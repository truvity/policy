// Package tenant is the checker for a company's configuration bundle: the
// directory of strict YAML sections a company hands a shared platform, the
// rules the bundle must satisfy on its own, the rules it must satisfy against
// what the platform holds about the company, and the digest that pins it.
//
// The contract is docs/contracts/tenant.md and the shapes are
// schemas/tenant.v1/. A refusal is all-or-nothing and aggregated: [Load],
// [Bundle.Validate] and [Bundle.Check] each report every failure they find as
// [Errors], each carrying a stable [Code], the file, the path of the row and a
// reason, and the caller keeps the last bundle it accepted.
//
// The package knows no company and no platform. What the platform holds about
// a company is a [Registration]; what it holds about everyone else is an
// [Estate] the caller implements.
package tenant

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Interface is the revision of the bundle format this package reads, and
// OldestInterface the oldest it still reads: a checker reads its own interface
// and the one before it. Interface 1 is the first, so both are 1.
const (
	Interface       = 1
	OldestInterface = 1
)

// Schema is the value of the manifest's `schema` key.
const Schema = "tenant.v1"

// Code is the stable name of a refusal. The codes are the interface: a message
// may be reworded, and a code is renamed only by a new interface number.
type Code string

const (
	// UnknownField means a field, or a file, that the contract does not name.
	UnknownField Code = "unknown-field"
	// Invalid means a value of the wrong type or shape, or a name that is not a
	// bare name.
	Invalid Code = "invalid"
	// UnknownInterface means the manifest names an interface newer than this
	// package reads, or a bundle format other than [Schema].
	UnknownInterface Code = "unknown-interface"
	// Duplicate means two rows claim one name, or a name the platform already gave
	// to another company.
	Duplicate Code = "duplicate"
	// MissingReason means a grant carries no `why`.
	MissingReason Code = "missing-reason"
	// CrossCompanyRef means a row refers to something the company does not own:
	// another company's project, or a project, group, receiver or endpoint
	// the bundle does not define. Also a manifest naming another company.
	CrossCompanyRef Code = "cross-company-ref"
	// ProviderOwned means a row takes a name the platform reserves for itself.
	ProviderOwned Code = "provider-owned"
	// CeilingExceeded means a row asks for more than the registration allows: a
	// zone, an e-mail domain or a registry the company does not hold, or a
	// request a domain's ceiling refuses.
	CeilingExceeded Code = "ceiling-exceeded"
	// WrongOrg means a source-hosting row names an organisation other than the
	// registered one.
	WrongOrg Code = "wrong-org"
)

// Codes lists every [Code], so that a test can prove each has a fixture.
func Codes() []Code {
	return []Code{
		UnknownField, Invalid, UnknownInterface, Duplicate, MissingReason,
		CrossCompanyRef, ProviderOwned, CeilingExceeded, WrongOrg,
	}
}

// Error is one refusal.
type Error struct {
	// Company is the manifest's company, or empty when the manifest itself
	// could not be read.
	Company string
	// File is the bundle file, such as "access.yaml".
	File string
	// Path locates the row within the file, such as "grants[2].scope". It is
	// empty when the whole file is at fault.
	Path string
	Code Code
	// Reason says what is wrong. It never contains a value from the bundle
	// beyond a name: an error is logged.
	Reason string
}

// Error formats as
//
//	tenant acme: access.yaml: grants[2]: scope "admin" exceeds ceiling "read" [ceiling-exceeded]
func (e Error) Error() string {
	var b strings.Builder

	b.WriteString("tenant")

	if e.Company != "" {
		b.WriteString(" " + e.Company)
	}

	b.WriteString(":")

	if e.File != "" {
		b.WriteString(" " + e.File + ":")
	}

	if e.Path != "" {
		b.WriteString(" " + e.Path + ":")
	}

	fmt.Fprintf(&b, " %s [%s]", e.Reason, e.Code)

	return b.String()
}

// Errors is every refusal found in one pass.
type Errors []Error

func (e Errors) Error() string {
	lines := make([]string, len(e))
	for i := range e {
		lines[i] = e[i].Error()
	}

	return strings.Join(lines, "\n")
}

// Has reports whether any refusal carries the code.
func (e Errors) Has(c Code) bool {
	for i := range e {
		if e[i].Code == c {
			return true
		}
	}

	return false
}

type (
	// Bundle is a company's configuration after [Load]. Every list is non-nil:
	// an omitted section and an empty one mean the same, no rows.
	Bundle struct {
		Manifest Manifest
		Projects []Project
		Clients  []Client
		Access   Access
		Alerts   Alerts
		Status   []StatusEntity
		CI       []CIRequest
		Registry []RegistryRow
		// GitHub is nil when the bundle has no github.yaml.
		GitHub *GitHub
	}

	// Manifest is tenant.yaml.
	Manifest struct {
		Schema    string `yaml:"schema"`
		Company   string `yaml:"company"`
		Interface int    `yaml:"interface"`
	}

	// Project is a project and the endpoints it publishes.
	Project struct {
		Name         string     `yaml:"name"`
		Description  string     `yaml:"description,omitempty"`
		Environments []string   `yaml:"environments,omitempty"`
		Endpoints    []Endpoint `yaml:"endpoints,omitempty"`
	}

	// Endpoint is one published host of a project.
	Endpoint struct {
		Name       string `yaml:"name"`
		Host       string `yaml:"host"`
		Visibility string `yaml:"visibility,omitempty"`
	}

	// Client is an OIDC client of one of the company's projects.
	Client struct {
		Name                   string   `yaml:"name"`
		Project                string   `yaml:"project"`
		Kind                   string   `yaml:"kind"`
		RedirectURIs           []string `yaml:"redirect_uris,omitempty"`
		PostLogoutRedirectURIs []string `yaml:"post_logout_redirect_uris,omitempty"`
	}

	// Access is access.yaml: groups, and the grants those groups hold.
	Access struct {
		Groups []Group `yaml:"groups"`
		Grants []Grant `yaml:"grants"`
	}

	// Group is a named set of people.
	Group struct {
		Name    string   `yaml:"name"`
		Members []string `yaml:"members,omitempty"`
	}

	// Grant is one request for access. It is a request, bounded by the
	// company's ceiling; it is not the access itself.
	Grant struct {
		Group   string `yaml:"group"`
		Project string `yaml:"project"`
		Scope   string `yaml:"scope"`
		// Why is required: a grant with no reason is the shape a stale grant
		// takes ten months later. It is enforced by [Bundle.Validate], not by
		// the schema, so that its absence has its own code.
		Why string `yaml:"why,omitempty"`
	}

	// Alerts is alerts.yaml: receivers, and the routes that reach them.
	Alerts struct {
		Receivers []Receiver `yaml:"receivers"`
		Routes    []Route    `yaml:"routes"`
	}

	// Receiver is where alerts go. Target is an e-mail address for kind
	// "email" and the NAME of a secret for kind "webhook".
	Receiver struct {
		Name   string `yaml:"name"`
		Kind   string `yaml:"kind"`
		Target string `yaml:"target"`
	}

	// Route sends one project's alerts to a receiver.
	Route struct {
		Project  string `yaml:"project"`
		Receiver string `yaml:"receiver"`
		Severity string `yaml:"severity,omitempty"`
	}

	// StatusEntity is an entity shown on the platform's status page.
	StatusEntity struct {
		Name     string `yaml:"name"`
		Project  string `yaml:"project"`
		Title    string `yaml:"title"`
		Endpoint string `yaml:"endpoint,omitempty"`
	}

	// CIRequest asks the platform's CI to run for a project.
	CIRequest struct {
		Name         string   `yaml:"name"`
		Project      string   `yaml:"project"`
		Repository   string   `yaml:"repository"`
		Environments []string `yaml:"environments,omitempty"`
	}

	// RegistryRow is a container-registry project row the platform hosts.
	RegistryRow struct {
		Project      string   `yaml:"project"`
		Registry     string   `yaml:"registry"`
		Repositories []string `yaml:"repositories,omitempty"`
	}

	// GitHub is github.yaml. Only the organisation is checked here; Rest
	// holds the other keys unread, for the tool that applies them.
	GitHub struct {
		Organization string         `yaml:"organization"`
		Rest         map[string]any `yaml:",inline"`
	}
)

// Registration is what the platform holds about one company. It is an input to
// [Bundle.Check] and never part of a bundle: a company cannot widen its own
// ceiling by editing its own repository.
type Registration struct {
	// Slug is the company's name; the manifest must carry it.
	Slug string
	// GitHubOrg is the source-hosting organisation, and GitHubOrgID its
	// numeric id. [Bundle.Check] compares the name; the id is carried for an
	// [Estate] that verifies it against the provider.
	GitHubOrg   string
	GitHubOrgID int64
	// EmailDomains are the domains the company's people sign in with.
	EmailDomains []string
	// Tailnet is the company's network name, carried for an [Estate].
	Tailnet string
	// Zones are the DNS zones the company owns, as hosts are written: lower
	// case, without a trailing dot. [Bundle.Check] reads a zone case- and
	// dot-insensitively, but hands an [Estate] the normalised spelling.
	Zones []string
	// Registries are the container registries the company may publish to.
	Registries []string
	// Ceilings holds one document per domain the company may delegate in,
	// keyed by domain ("access" today). They are opaque here: each domain's
	// library defines its own ceiling and enforces it through
	// [Estate.WithinCeiling]. An omitted ceiling admits nothing. Only the
	// [AccessDomain] ceiling is enforced in interface 1: a ceiling for any
	// other domain is carried and ignored, without a refusal.
	Ceilings map[string]json.RawMessage
}

// Kinds of reserved name an [Estate] is asked about.
const (
	KindProject  = "project"
	KindGroup    = "group"
	KindClient   = "client"
	KindReceiver = "receiver"
	KindHost     = "host"
)

// AccessDomain is the key of the access ceiling in [Registration.Ceilings].
const AccessDomain = "access"

// Estate is what the platform knows about everyone else. The caller
// implements it over whatever it keeps its companies in; this package knows no
// platform.
type Estate interface {
	// ProjectOwner says which company holds a project name.
	ProjectOwner(project string) (company string, ok bool)
	// ProviderOwned says the platform reserves a name of the given kind
	// (one of the Kind constants) for itself.
	ProviderOwned(kind, name string) bool
	// ZoneOwner says which company owns a DNS zone.
	ZoneOwner(zone string) (company string, ok bool)
	// WithinCeiling checks one row against a domain's ceiling, which this
	// package does not read. A non-nil error refuses the row, and its text is
	// the reason. For the access domain the row is a [Grant].
	WithinCeiling(domain string, ceiling json.RawMessage, row any) error
}
