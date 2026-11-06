package tenant

import (
	"regexp"
	"strings"
)

var (
	// nameRE is what a project, group, client, receiver, endpoint or entity
	// name may look like: one name, never a glob.
	nameRE = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

	// emailRE is the schema's e-mail form: one address, never a pattern.
	emailRE = regexp.MustCompile(`^[A-Za-z0-9._%+-]+@[a-z0-9]([a-z0-9.-]*[a-z0-9])?$`)

	hostRE = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]*[a-z0-9])?\.)+[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
)

// Validate checks everything that needs no platform: that every name is one
// bare name, that nothing is claimed twice, that every grant has a reason, and
// that every row refers only to rows the bundle itself defines.
//
// It does not re-check the shape [Load] checks (enums, URL forms): a bundle
// built in code is checked for its names, reasons, uniqueness and references,
// which is what a bundle read from files can no longer get wrong silently.
//
// It reports every failure it finds, as [Errors].
func (b *Bundle) Validate() error {
	c := &collector{company: b.Manifest.Company}
	b.validate(c)

	return c.result()
}

// seen records the names a list has used, and reports a repeat.
type seen map[string]bool

func (s seen) again(key string) bool {
	if s[key] {
		return true
	}

	s[key] = true

	return false
}

func (b *Bundle) validate(c *collector) {
	b.validateManifest(c)

	projects := b.validateProjects(c)

	for i := range b.Clients {
		b.validateClient(c, i, projects)
	}

	b.validateAccess(c, projects)
	b.validateAlerts(c, projects)
	b.validateStatus(c, projects)
	b.validateCI(c, projects)
	b.validateRegistry(c, projects)

	if b.GitHub != nil && strings.TrimSpace(b.GitHub.Organization) == "" {
		c.add("github.yaml", "organization", Invalid, "the organisation is required")
	}
}

func (b *Bundle) validateManifest(c *collector) {
	m := b.Manifest

	if m.Schema != Schema {
		c.add(manifestFile, "schema", UnknownInterface, "not a bundle format this checker reads (it reads %s)", Schema)
	}

	if m.Interface < OldestInterface || m.Interface > Interface {
		c.add(manifestFile, "interface", UnknownInterface,
			"interface %d is not one this checker reads (%d to %d)", m.Interface, OldestInterface, Interface)
	}

	if !nameRE.MatchString(m.Company) {
		c.add(manifestFile, "company", Invalid, "the company is not one bare name")
	}
}

// name refuses a value that is not one bare name.
func (c *collector) name(file, path, value string) {
	if !nameRE.MatchString(value) {
		c.add(file, path, Invalid, "not one bare name (lower case letters, digits and hyphens; never a pattern)")
	}
}

// ref refuses a reference to a row the bundle does not define.
func (c *collector) ref(file, path, what, value string, defined map[string]bool) {
	if !defined[value] {
		c.add(file, path, CrossCompanyRef, "%s %q is not defined in this bundle, so it is not this company's to refer to", what, value)
	}
}

// validateProjects returns the project names, and for each the names of its
// endpoints.
func (b *Bundle) validateProjects(c *collector) map[string]map[string]bool {
	const file = "projects.yaml"

	projects := map[string]map[string]bool{}
	names := seen{}
	hosts := seen{}

	for i, p := range b.Projects {
		c.name(file, at("projects", i)+".name", p.Name)

		if names.again(p.Name) {
			c.add(file, at("projects", i)+".name", Duplicate, "project %q is defined twice", p.Name)
		}

		endpoints := map[string]bool{}
		epNames := seen{}

		for j, e := range p.Endpoints {
			path := at("projects", i) + "." + at("endpoints", j)

			c.name(file, path+".name", e.Name)

			if epNames.again(e.Name) {
				c.add(file, path+".name", Duplicate, "endpoint %q is defined twice in project %q", e.Name, p.Name)
			}

			if !hostRE.MatchString(e.Host) {
				c.add(file, path+".host", Invalid, "not one DNS name (never a wildcard)")
			}

			if hosts.again(e.Host) {
				c.add(file, path+".host", Duplicate, "the host %q is claimed by more than one endpoint", e.Host)
			}

			endpoints[e.Name] = true
		}

		if _, dup := projects[p.Name]; !dup {
			projects[p.Name] = endpoints
		}
	}

	return projects
}

func definedProjects(projects map[string]map[string]bool) map[string]bool {
	out := make(map[string]bool, len(projects))
	for name := range projects {
		out[name] = true
	}

	return out
}

func (b *Bundle) validateClient(c *collector, i int, projects map[string]map[string]bool) {
	const file = "clients.yaml"

	cl := b.Clients[i]
	path := at("clients", i)

	c.name(file, path+".name", cl.Name)

	for j := 0; j < i; j++ {
		if b.Clients[j].Name == cl.Name {
			c.add(file, path+".name", Duplicate, "client %q is defined twice", cl.Name)
			break
		}
	}

	c.ref(file, path+".project", "project", cl.Project, definedProjects(projects))
}

func (b *Bundle) validateAccess(c *collector, projects map[string]map[string]bool) {
	const file = "access.yaml"

	groups := map[string]bool{}
	groupNames := seen{}

	for i, g := range b.Access.Groups {
		path := at("groups", i)

		c.name(file, path+".name", g.Name)

		if groupNames.again(g.Name) {
			c.add(file, path+".name", Duplicate, "group %q is defined twice", g.Name)
		}

		groups[g.Name] = true

		members := seen{}

		for j, m := range g.Members {
			mp := path + "." + at("members", j)

			if strings.ContainsAny(m, "*?") || !strings.Contains(m, "@") {
				c.add(file, mp, Invalid, "not one e-mail address (never a pattern)")
			}

			if members.again(strings.ToLower(m)) {
				c.add(file, mp, Duplicate, "listed twice in group %q", g.Name)
			}
		}
	}

	known := definedProjects(projects)
	pairs := seen{}

	for i, g := range b.Access.Grants {
		path := at("grants", i)

		c.ref(file, path+".group", "group", g.Group, groups)
		c.ref(file, path+".project", "project", g.Project, known)

		if pairs.again(g.Group + "\x00" + g.Project) {
			c.add(file, path, Duplicate, "group %q already holds a grant on project %q", g.Group, g.Project)
		}

		if strings.TrimSpace(g.Why) == "" {
			c.add(file, path+".why", MissingReason, "a grant needs a reason: one with none is the shape a stale grant takes")
		}
	}
}

func (b *Bundle) validateAlerts(c *collector, projects map[string]map[string]bool) {
	const file = "alerts.yaml"

	receivers := map[string]bool{}
	names := seen{}

	for i, r := range b.Alerts.Receivers {
		path := at("receivers", i)

		c.name(file, path+".name", r.Name)

		if names.again(r.Name) {
			c.add(file, path+".name", Duplicate, "receiver %q is defined twice", r.Name)
		}

		switch r.Kind {
		case "webhook":
			if !nameRE.MatchString(r.Target) {
				c.add(file, path+".target", Invalid, "a webhook names the secret that holds its address (one bare name), never the address")
			}
		case "email":
			if !emailRE.MatchString(r.Target) {
				c.add(file, path+".target", Invalid, "not one e-mail address (never a pattern)")
			}
		}

		receivers[r.Name] = true
	}

	known := definedProjects(projects)
	routes := seen{}

	for i, r := range b.Alerts.Routes {
		path := at("routes", i)

		c.ref(file, path+".project", "project", r.Project, known)
		c.ref(file, path+".receiver", "receiver", r.Receiver, receivers)

		if routes.again(r.Project + "\x00" + r.Receiver + "\x00" + r.Severity) {
			c.add(file, path, Duplicate, "the same route is listed twice")
		}
	}
}

func (b *Bundle) validateStatus(c *collector, projects map[string]map[string]bool) {
	const file = "status.yaml"

	known := definedProjects(projects)
	names := seen{}

	for i, e := range b.Status {
		path := at("entities", i)

		c.name(file, path+".name", e.Name)

		if names.again(e.Name) {
			c.add(file, path+".name", Duplicate, "entity %q is defined twice", e.Name)
		}

		c.ref(file, path+".project", "project", e.Project, known)

		if e.Endpoint != "" && known[e.Project] && !projects[e.Project][e.Endpoint] {
			c.add(file, path+".endpoint", CrossCompanyRef, "endpoint %q is not an endpoint of project %q in this bundle", e.Endpoint, e.Project)
		}
	}
}

func (b *Bundle) validateCI(c *collector, projects map[string]map[string]bool) {
	const file = "ci.yaml"

	known := definedProjects(projects)
	names := seen{}

	for i, r := range b.CI {
		path := at("requests", i)

		c.name(file, path+".name", r.Name)

		if names.again(r.Name) {
			c.add(file, path+".name", Duplicate, "request %q is defined twice", r.Name)
		}

		c.ref(file, path+".project", "project", r.Project, known)
	}
}

func (b *Bundle) validateRegistry(c *collector, projects map[string]map[string]bool) {
	const file = "registry.yaml"

	known := definedProjects(projects)
	rows := seen{}

	for i, r := range b.Registry {
		path := at("projects", i)

		c.ref(file, path+".project", "project", r.Project, known)

		if rows.again(r.Project + "\x00" + r.Registry) {
			c.add(file, path, Duplicate, "project %q already has a row for registry %q", r.Project, r.Registry)
		}
	}
}
