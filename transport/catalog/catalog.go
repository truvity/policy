// Package catalog is the who-calls-whom catalog for workload mutual TLS: for
// each callee (a component of a project's chart that exposes
// `tls.peers.<component>`, see the `tls` fragment) the identities outside its
// own release that may call it, per environment.
//
// It is grants only. A chart grants its own release's account to itself,
// because it wired the call; duplicating that here could only drift.
//
// The package owns the file format (schemas/calls.json), the rules a catalog
// must satisfy on its own, the derivation of each release's `tls.peers`
// values, and the cross-checks against an estate's projects. It knows no
// estate: the trust domain is a parameter, and the estate is an [Estate] the
// caller implements.
package catalog

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/truvity/policy"
	"github.com/truvity/policy/config"
	"github.com/truvity/policy/transport"
)

type (
	// Catalog is the whole file.
	Catalog struct {
		// Services is keyed "<project>/<component>".
		Services map[string]Service `yaml:"services"`
	}

	// Service is one callee.
	Service struct {
		// Environments is keyed by environment name.
		Environments map[string]ServiceEnvironment `yaml:"environments"`
		// GatewayFronted marks a component an edge terminates TLS for: it
		// can never be strict, because a strict one refuses its only caller.
		GatewayFronted bool `yaml:"gateway_fronted,omitempty"`
	}

	// ServiceEnvironment is one environment's grant for one service. Peers
	// is never nil after Parse, so an omitted list and an empty one render
	// alike (an omitted list renders null, which a service reading
	// `tls.peers` refuses).
	ServiceEnvironment struct {
		Peers []Peer `yaml:"peers"`
		// Strict marks the component to run tls strict. Whether it takes
		// effect is the estate's call: see [Estate.Enforced].
		Strict bool `yaml:"strict,omitempty"`
	}

	// Peer is one identity allowed to call a service from outside its release.
	// Exactly one of Project or Platform is set. A Project peer's namespace is
	// the project's own name, so it states none; a Platform peer, which has no
	// project to resolve from, states its Namespace.
	Peer struct {
		Project        string `yaml:"project,omitempty"`
		Platform       string `yaml:"platform,omitempty"`
		Namespace      string `yaml:"namespace,omitempty"`
		ServiceAccount string `yaml:"serviceAccount"`
		// Why is required: a grant with no reason is the shape a stale
		// grant takes ten months later.
		Why string `yaml:"why"`
	}

	// Estate is what the catalog is checked against. The caller implements it
	// over whatever it keeps its projects in.
	Estate interface {
		HasProject(project string) bool
		HasEnvironment(environment string) bool
		// HasIdentity says the project mints workload identities in the
		// environment. A grant naming a project that cannot present one, or
		// a callee that never renders `tls.peers` there, is refused.
		HasIdentity(project, environment string) bool
		// Enforced says the project's `strict` marks take effect there.
		Enforced(project, environment string) bool
	}
)

// nameRE is what a namespace, account, project or platform name may look
// like: one name, never a glob.
var nameRE = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// Parse reads a catalog: the document is checked against schemas/calls.json,
// decoded strictly, normalised and validated with [Catalog.Validate].
func Parse(data []byte) (*Catalog, error) {
	var doc any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("catalog: %w", err)
	}

	schema, err := policy.Schemas.ReadFile("schemas/calls.json")
	if err != nil {
		return nil, err
	}

	if err := config.Validate(doc, schema); err != nil {
		return nil, fmt.Errorf("catalog: %w", err)
	}

	var c Catalog

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	if err := dec.Decode(&c); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("catalog: %w", err)
	}

	c.normalize()

	if err := c.Validate(); err != nil {
		return nil, err
	}

	return &c, nil
}

func (c *Catalog) normalize() {
	for _, svc := range c.Services {
		for name, e := range svc.Environments {
			if e.Peers == nil {
				e.Peers = []Peer{}
				svc.Environments[name] = e
			}
		}
	}
}

// Validate checks everything that can be checked without an estate: the keys,
// every peer, and that no gateway-fronted component is strict.
func (c *Catalog) Validate() error {
	for _, key := range c.keys() {
		project, component, err := splitKey(key)
		if err != nil {
			return err
		}

		for _, name := range []string{project, component} {
			if !nameRE.MatchString(name) {
				return fmt.Errorf("catalog: services key %q: %q is not a bare name", key, name)
			}
		}

		svc := c.Services[key]

		for _, environment := range sortedKeys(svc.Environments) {
			env := svc.Environments[environment]

			mode := transport.Permissive
			if env.Strict {
				mode = transport.Strict
			}

			if err := transport.CheckMode(mode, svc.GatewayFronted); err != nil {
				return fmt.Errorf("catalog: %s/%s: %w", key, environment, err)
			}

			seen := map[string]bool{}

			for i := range env.Peers {
				peer := &env.Peers[i]
				if err := peer.validate(); err != nil {
					return fmt.Errorf("catalog: %s/%s: %w", key, environment, err)
				}

				id := peer.account()
				if seen[id] {
					return fmt.Errorf("catalog: %s/%s: peer %s is granted twice", key, environment, id)
				}

				seen[id] = true
			}
		}
	}

	return nil
}

func (p *Peer) validate() error {
	if (p.Project == "") == (p.Platform == "") {
		return errors.New("peer must name exactly one of project or platform")
	}

	label := p.label()

	if p.Why == "" {
		return fmt.Errorf("peer %s: why is required", label)
	}

	if p.ServiceAccount == "" {
		return fmt.Errorf("peer %s: serviceAccount is required", label)
	}

	if !nameRE.MatchString(p.ServiceAccount) {
		return fmt.Errorf("peer %s: serviceAccount %q is not a bare name", label, p.ServiceAccount)
	}

	if p.Project != "" {
		if !nameRE.MatchString(p.Project) {
			return fmt.Errorf("peer project %q is not a bare name", p.Project)
		}

		if p.Namespace != "" {
			return fmt.Errorf("peer %s: namespace is set on a project peer — a project's namespace is derived, never stated", label)
		}

		return nil
	}

	if !nameRE.MatchString(p.Platform) {
		return fmt.Errorf("peer platform %q is not a bare name", p.Platform)
	}

	if p.Namespace == "" {
		return fmt.Errorf("peer %s: namespace is required for a platform peer", label)
	}

	if !nameRE.MatchString(p.Namespace) {
		return fmt.Errorf("peer %s: namespace %q is not a bare name", label, p.Namespace)
	}

	return nil
}

func (p *Peer) label() string {
	if p.Project != "" {
		return "project:" + p.Project
	}

	return "platform:" + p.Platform
}

// account is the "<namespace>/<serviceAccount>" the peer resolves to.
func (p *Peer) account() string {
	ns, sa, _ := p.Resolve("")

	return ns + "/" + sa
}

// Resolve returns the namespace and account the peer renders as, and its
// SPIFFE identity in the given trust domain. A project peer's namespace is the
// project's own name.
func (p *Peer) Resolve(trustDomain string) (namespace, serviceAccount, spiffeID string) {
	namespace = p.Namespace
	if p.Project != "" {
		namespace = p.Project
	}

	return namespace, p.ServiceAccount, transport.SpiffeID(trustDomain, namespace, p.ServiceAccount)
}

// ValidateEstate binds the catalog to an estate: every service's project and
// environment must exist and mint identities there, every project peer must
// exist and mint identities in that environment, and where a project is
// enforced every strict component must grant every caller the catalog knows
// for that project there — a caller granted on one component but not on a
// strict sibling is refused at its handshake once strict takes effect.
func (c *Catalog) ValidateEstate(estate Estate) error {
	if estate == nil {
		return errors.New("catalog: an estate is required for cross-validation")
	}

	for _, key := range c.keys() {
		project, component, err := splitKey(key)
		if err != nil {
			return err
		}

		if !estate.HasProject(project) {
			return fmt.Errorf("catalog: %s: the estate has no project %q", key, project)
		}

		svc := c.Services[key]

		for _, environment := range sortedKeys(svc.Environments) {
			if !estate.HasEnvironment(environment) {
				return fmt.Errorf("catalog: %s: names unknown environment %q", key, environment)
			}

			if !estate.HasIdentity(project, environment) {
				return fmt.Errorf("catalog: %s: project %q has no workload identity in %q, so tls.peers.%s never renders there",
					key, project, environment, component)
			}

			for _, peer := range svc.Environments[environment].Peers {
				if peer.Project == "" {
					continue // not a project: nothing to look up
				}

				if !estate.HasProject(peer.Project) {
					return fmt.Errorf("catalog: %s/%s: peer names project %q, which the estate does not have", key, environment, peer.Project)
				}

				if !estate.HasIdentity(peer.Project, environment) {
					return fmt.Errorf("catalog: %s/%s: peer project %q has no workload identity in %q, so it can never present a certificate",
						key, environment, peer.Project, environment)
				}
			}
		}
	}

	return c.checkEnforced(estate)
}

func (c *Catalog) checkEnforced(estate Estate) error {
	for _, key := range c.keys() {
		project, component, _ := splitKey(key)

		for _, environment := range sortedKeys(c.Services[key].Environments) {
			if !c.Services[key].Environments[environment].Strict || !estate.Enforced(project, environment) {
				continue
			}

			granted := map[string]bool{}
			for _, peer := range c.PeersFor(project, component, environment) {
				granted[peer.account()] = true
			}

			for _, caller := range c.callers(project, environment) {
				if !granted[caller.account] {
					return fmt.Errorf("catalog: %s/%s: strict and enforced, but caller %s (granted on %s) has no grant on %s — it would be refused at the handshake",
						key, environment, caller.account, caller.where, key)
				}
			}
		}
	}

	return nil
}

type caller struct{ account, where string }

// callers lists every distinct caller of a project in an environment — the
// union of the peers on all of its rows there — sorted, with the first row
// naming each.
func (c *Catalog) callers(project, environment string) []caller {
	first := map[string]string{}

	for _, key := range c.keys() {
		p, component, err := splitKey(key)
		if err != nil || p != project {
			continue
		}

		for _, peer := range c.PeersFor(project, component, environment) {
			if _, ok := first[peer.account()]; !ok {
				first[peer.account()] = key
			}
		}
	}

	out := make([]caller, 0, len(first))
	for _, account := range sortedKeys(first) {
		out = append(out, caller{account: account, where: first[account]})
	}

	return out
}

// PeersFor returns the peers declared for one service in one environment, or
// nil when the catalog never mentions the pair — the signal to render nothing
// and leave the chart's own default in force. A declared-but-empty
// environment returns a non-nil empty slice.
func (c *Catalog) PeersFor(project, component, environment string) []Peer {
	env, ok := c.Services[project+"/"+component].Environments[environment]
	if !ok {
		return nil
	}

	return env.Peers
}

// TLSPeers derives a release's `tls.peers.<component>` values for one
// environment: component → the accounts granted. A component the catalog does
// not mention in the environment has no key; nil when none has.
func (c *Catalog) TLSPeers(project, environment string) map[string][]transport.Peer {
	var out map[string][]transport.Peer

	for _, key := range c.keys() {
		p, component, err := splitKey(key)
		if err != nil || p != project {
			continue
		}

		peers := c.PeersFor(project, component, environment)
		if peers == nil {
			continue
		}

		if out == nil {
			out = map[string][]transport.Peer{}
		}

		resolved := make([]transport.Peer, 0, len(peers))
		for i := range peers {
			ns, sa, _ := peers[i].Resolve("")
			resolved = append(resolved, transport.Peer{Namespace: ns, ServiceAccount: sa})
		}

		out[component] = resolved
	}

	return out
}

// StrictComponents lists, sorted, the components marked strict for a project in
// an environment. It does not ask whether strict takes effect; that is
// [Estate.Enforced].
func (c *Catalog) StrictComponents(project, environment string) []string {
	var out []string

	for _, key := range c.keys() {
		p, component, err := splitKey(key)
		if err == nil && p == project && c.Services[key].Environments[environment].Strict {
			out = append(out, component)
		}
	}

	return out
}

func splitKey(key string) (project, component string, err error) {
	project, component, ok := strings.Cut(key, "/")
	if !ok {
		return "", "", fmt.Errorf("catalog: services key %q must be \"<project>/<component>\"", key)
	}

	return project, component, nil
}

func (c *Catalog) keys() []string { return sortedKeys(c.Services) }

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}
