package tenant

import (
	"encoding/json"
	"errors"
	"strings"
)

// Check validates the bundle ([Bundle.Validate]) and then checks it against
// what the platform holds: the registration of the company and, for everyone
// else, the estate.
//
// A bundle is refused for: a manifest naming another company; a project name
// another company holds; a name the platform reserves; an endpoint host
// outside the zones the company owns, or in a zone another company owns; a
// member or an e-mail receiver outside the company's e-mail domains; a
// registry the company does not hold; a repository or organisation other than
// the registered one; and a grant its access ceiling refuses. An omitted
// ceiling admits nothing, so a bundle with grants and no access ceiling is
// refused row by row.
//
// A nil estate owns nothing, reserves nothing and holds no ceiling: it admits
// a bundle that asks for nothing that needs the platform's say.
//
// Every failure is reported, as [Errors]; nothing is accepted in part.
func (b *Bundle) Check(reg Registration, est Estate) error {
	if est == nil {
		est = noEstate{}
	}

	c := &collector{company: b.Manifest.Company}
	b.validate(c)

	if b.Manifest.Company != reg.Slug {
		c.add(manifestFile, "company", CrossCompanyRef, "the bundle is for another company than the one registered")
	}

	b.checkProjects(c, reg, est)
	b.checkClients(c, est)
	b.checkAccess(c, reg, est)
	b.checkAlerts(c, reg, est)
	b.checkRegistry(c, reg)
	b.checkSourceHosting(c, reg)

	return c.result()
}

func (b *Bundle) checkProjects(c *collector, reg Registration, est Estate) {
	const file = "projects.yaml"

	for i, p := range b.Projects {
		path := at("projects", i)

		if est.ProviderOwned(KindProject, p.Name) {
			c.add(file, path+".name", ProviderOwned, "the name %q is reserved by the platform", p.Name)
		}

		if owner, ok := est.ProjectOwner(p.Name); ok && owner != reg.Slug {
			c.add(file, path+".name", Duplicate, "the name %q is already held by another company", p.Name)
		}

		for j, e := range p.Endpoints {
			hp := path + "." + at("endpoints", j) + ".host"

			if est.ProviderOwned(KindHost, e.Host) {
				c.add(file, hp, ProviderOwned, "the host %q is reserved by the platform", e.Host)
			}

			zone, ok := zoneOf(e.Host, reg.Zones)
			if !ok {
				c.add(file, hp, CeilingExceeded, "the host %q is outside every zone the company owns", e.Host)
				continue
			}

			if owner, held := est.ZoneOwner(zone); held && owner != reg.Slug {
				c.add(file, hp, CeilingExceeded, "the zone %q is owned by another company", zone)
			}
		}
	}
}

// zoneOf finds the longest zone the host is in: the zone itself, or a name
// below it. "evilacme.example" is not below "acme.example".
func zoneOf(host string, zones []string) (string, bool) {
	best := ""

	for _, z := range zones {
		z = strings.TrimSuffix(strings.ToLower(z), ".")
		if (host == z || strings.HasSuffix(host, "."+z)) && len(z) > len(best) {
			best = z
		}
	}

	return best, best != ""
}

func (b *Bundle) checkClients(c *collector, est Estate) {
	for i, cl := range b.Clients {
		if est.ProviderOwned(KindClient, cl.Name) {
			c.add("clients.yaml", at("clients", i)+".name", ProviderOwned, "the name %q is reserved by the platform", cl.Name)
		}
	}
}

func (b *Bundle) checkAccess(c *collector, reg Registration, est Estate) {
	const file = "access.yaml"

	for i, g := range b.Access.Groups {
		path := at("groups", i)

		if est.ProviderOwned(KindGroup, g.Name) {
			c.add(file, path+".name", ProviderOwned, "the name %q is reserved by the platform", g.Name)
		}

		for j, m := range g.Members {
			if !inDomains(m, reg.EmailDomains) {
				c.add(file, path+"."+at("members", j), CeilingExceeded, "the address is outside the e-mail domains the company holds")
			}
		}
	}

	ceiling, held := reg.Ceilings[AccessDomain]

	for i, g := range b.Access.Grants {
		path := at("grants", i)

		if !held {
			c.add(file, path, CeilingExceeded, "the company has no access ceiling, and an omitted ceiling admits nothing")
			continue
		}

		if err := est.WithinCeiling(AccessDomain, ceiling, g); err != nil {
			c.add(file, path, CeilingExceeded, "%s", err.Error())
		}
	}
}

func inDomains(address string, domains []string) bool {
	_, domain, ok := strings.Cut(address, "@")
	if !ok {
		return false
	}

	for _, d := range domains {
		if strings.EqualFold(domain, d) {
			return true
		}
	}

	return false
}

func (b *Bundle) checkAlerts(c *collector, reg Registration, est Estate) {
	for i, r := range b.Alerts.Receivers {
		path := at("receivers", i)

		if est.ProviderOwned(KindReceiver, r.Name) {
			c.add("alerts.yaml", path+".name", ProviderOwned, "the name %q is reserved by the platform", r.Name)
		}

		if r.Kind == "email" && !inDomains(r.Target, reg.EmailDomains) {
			c.add("alerts.yaml", path+".target", CeilingExceeded, "the address is outside the e-mail domains the company holds")
		}
	}
}

func (b *Bundle) checkRegistry(c *collector, reg Registration) {
	held := map[string]bool{}
	for _, r := range reg.Registries {
		held[r] = true
	}

	for i, r := range b.Registry {
		if !held[r.Registry] {
			c.add("registry.yaml", at("projects", i)+".registry", CeilingExceeded, "the registry %q is not one the company holds", r.Registry)
		}
	}
}

func (b *Bundle) checkSourceHosting(c *collector, reg Registration) {
	for i, r := range b.CI {
		owner, _, _ := strings.Cut(r.Repository, "/")
		if !strings.EqualFold(owner, reg.GitHubOrg) {
			c.add("ci.yaml", at("requests", i)+".repository", WrongOrg, "the repository is not in the company's registered organisation")
		}
	}

	if b.GitHub != nil && !strings.EqualFold(b.GitHub.Organization, reg.GitHubOrg) {
		c.add("github.yaml", "organization", WrongOrg, "not the company's registered organisation")
	}
}

// noEstate is the platform a nil [Estate] stands for.
type noEstate struct{}

func (noEstate) ProjectOwner(string) (string, bool) { return "", false }
func (noEstate) ProviderOwned(_, _ string) bool     { return false }
func (noEstate) ZoneOwner(string) (string, bool)    { return "", false }
func (noEstate) WithinCeiling(string, json.RawMessage, any) error {
	return errors.New("there is no estate to read the ceiling")
}
