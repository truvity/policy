package catalog_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/truvity/policy/transport"
	"github.com/truvity/policy/transport/catalog"
)

const sample = `
services:
  shop/api:
    environments:
      dev:
        peers:
          - project: shop
            serviceAccount: shop-prober
            why: the prober calls the API
      prod:
        peers: []
  shop/web:
    gateway_fronted: true
    environments:
      dev: {}
`

func single(peer catalog.Peer) *catalog.Catalog {
	return &catalog.Catalog{Services: map[string]catalog.Service{
		"shop/api": {Environments: map[string]catalog.ServiceEnvironment{
			"dev": {Peers: []catalog.Peer{peer}},
		}},
	}}
}

func TestParseNormalisesAndResolves(t *testing.T) {
	c, err := catalog.Parse([]byte(sample))
	require.NoError(t, err)

	assert.Empty(t, c.PeersFor("shop", "api", "prod"))
	assert.NotNil(t, c.PeersFor("shop", "api", "prod"), "declared and empty is not unmentioned")
	assert.NotNil(t, c.PeersFor("shop", "web", "dev"), "an omitted peers list is an empty one")
	assert.Nil(t, c.PeersFor("shop", "api", "stage"), "unmentioned environment")
	assert.Nil(t, c.PeersFor("shop", "worker", "dev"), "unmentioned service")

	peers := c.PeersFor("shop", "api", "dev")
	require.Len(t, peers, 1)

	ns, sa, id := peers[0].Resolve("dev.example.internal")
	assert.Equal(t, "shop", ns)
	assert.Equal(t, "shop-prober", sa)
	assert.Equal(t, "spiffe://dev.example.internal/ns/shop/sa/shop-prober", id)
}

func TestParseRefusesWhatTheSchemaRefuses(t *testing.T) {
	for name, doc := range map[string]string{
		"unknown key":  "services: {}\nextra: 1\n",
		"no services":  "{}\n",
		"bad key":      "services:\n  shop:\n    environments: {}\n",
		"missing why":  "services:\n  shop/api:\n    environments:\n      dev:\n        peers:\n          - {project: shop, serviceAccount: x}\n",
		"glob account": "services:\n  shop/api:\n    environments:\n      dev:\n        peers:\n          - {project: shop, serviceAccount: '*', why: w}\n",
		"both kinds":   "services:\n  shop/api:\n    environments:\n      dev:\n        peers:\n          - {project: shop, platform: gw, namespace: n, serviceAccount: x, why: w}\n",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := catalog.Parse([]byte(doc))
			assert.Error(t, err)
		})
	}
}

func TestTLSPeersIsTheReleasesValues(t *testing.T) {
	c, err := catalog.Parse([]byte(sample))
	require.NoError(t, err)

	assert.Equal(t, map[string][]transport.Peer{
		"api": {{Namespace: "shop", ServiceAccount: "shop-prober"}},
		"web": {},
	}, c.TLSPeers("shop", "dev"))
	assert.Equal(t, map[string][]transport.Peer{"api": {}}, c.TLSPeers("shop", "prod"))
	assert.Nil(t, c.TLSPeers("shop", "stage"))
	assert.Nil(t, c.TLSPeers("other", "dev"))
}

func TestValidateKeys(t *testing.T) {
	for _, tc := range []struct{ name, key string }{
		{"no slash", "shop"},
		{"empty component", "shop/"},
		{"empty project", "/api"},
		{"glob", "sh*p/api"},
		{"upper case", "Shop/api"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &catalog.Catalog{Services: map[string]catalog.Service{
				tc.key: {Environments: map[string]catalog.ServiceEnvironment{"dev": {Peers: []catalog.Peer{}}}},
			}}
			assert.Error(t, c.Validate())
		})
	}
}

func TestValidatePeers(t *testing.T) {
	ok := catalog.Peer{Project: "shop", ServiceAccount: "x", Why: "w"}
	platform := catalog.Peer{Platform: "gateway", Namespace: "edge", ServiceAccount: "gw", Why: "w"}

	assert.NoError(t, single(ok).Validate())
	assert.NoError(t, single(platform).Validate())

	for name, peer := range map[string]catalog.Peer{
		"neither project nor platform": {ServiceAccount: "x", Why: "w"},
		"both":                         {Project: "shop", Platform: "gw", Namespace: "n", ServiceAccount: "x", Why: "w"},
		"no why":                       {Project: "shop", ServiceAccount: "x"},
		"no account":                   {Project: "shop", Why: "w"},
		"platform without namespace":   {Platform: "gw", ServiceAccount: "x", Why: "w"},
		"project with namespace":       {Project: "shop", Namespace: "shop", ServiceAccount: "x", Why: "w"},
		"glob in account":              {Project: "shop", ServiceAccount: "*", Why: "w"},
		"glob in project":              {Project: "sh*", ServiceAccount: "x", Why: "w"},
		"glob in platform":             {Platform: "g*", Namespace: "n", ServiceAccount: "x", Why: "w"},
		"glob in namespace":            {Platform: "gw", Namespace: "n*", ServiceAccount: "x", Why: "w"},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Error(t, single(peer).Validate())
		})
	}
}

func TestValidateRefusesADuplicateGrant(t *testing.T) {
	p := catalog.Peer{Project: "shop", ServiceAccount: "x", Why: "w"}
	c := single(p)
	env := c.Services["shop/api"].Environments["dev"]
	env.Peers = append(env.Peers, p)
	c.Services["shop/api"].Environments["dev"] = env

	assert.ErrorContains(t, c.Validate(), "twice")
}

func TestStrictNeverOnAGatewayFrontedComponent(t *testing.T) {
	c := &catalog.Catalog{Services: map[string]catalog.Service{
		"shop/web": {GatewayFronted: true, Environments: map[string]catalog.ServiceEnvironment{
			"dev": {Peers: []catalog.Peer{}, Strict: true},
		}},
	}}

	assert.ErrorContains(t, c.Validate(), "gateway-fronted")
}

// estate is a map of project → environment → state.
type estate map[string]map[string]string

func (e estate) HasProject(p string) bool { _, ok := e[p]; return ok }
func (e estate) HasEnvironment(env string) bool {
	return env == "dev" || env == "prod"
}
func (e estate) HasIdentity(p, env string) bool { return e[p][env] != "" }
func (e estate) Enforced(p, env string) bool    { return e[p][env] == "enforced" }

func TestValidateEstate(t *testing.T) {
	good := estate{"shop": {"dev": "identity", "prod": "identity"}, "legacy": {}}
	prober := catalog.Peer{Project: "shop", ServiceAccount: "shop-prober", Why: "w"}

	c := single(prober)
	assert.NoError(t, c.ValidateEstate(good))
	assert.Error(t, c.ValidateEstate(nil))

	t.Run("unknown project", func(t *testing.T) {
		assert.ErrorContains(t, c.ValidateEstate(estate{}), "no project")
	})
	t.Run("unknown environment", func(t *testing.T) {
		bad := &catalog.Catalog{Services: map[string]catalog.Service{
			"shop/api": {Environments: map[string]catalog.ServiceEnvironment{"moon": {Peers: []catalog.Peer{}}}},
		}}
		assert.ErrorContains(t, bad.ValidateEstate(good), "unknown environment")
	})
	t.Run("callee without identity", func(t *testing.T) {
		assert.ErrorContains(t, c.ValidateEstate(estate{"shop": {"prod": "identity"}}), "no workload identity")
	})
	t.Run("peer project unknown", func(t *testing.T) {
		assert.ErrorContains(t, single(catalog.Peer{Project: "ghost", ServiceAccount: "x", Why: "w"}).ValidateEstate(good), "does not have")
	})
	t.Run("peer project without identity", func(t *testing.T) {
		assert.ErrorContains(t, single(catalog.Peer{Project: "legacy", ServiceAccount: "x", Why: "w"}).ValidateEstate(good), "no workload identity")
	})
	t.Run("platform peer needs no project", func(t *testing.T) {
		assert.NoError(t, single(catalog.Peer{Platform: "gateway", Namespace: "edge", ServiceAccount: "gw", Why: "w"}).ValidateEstate(good))
	})
}

func TestEnforcedStrictNeedsEveryKnownCallerGranted(t *testing.T) {
	prober := catalog.Peer{Project: "shop", ServiceAccount: "shop-prober", Why: "w"}
	c := &catalog.Catalog{Services: map[string]catalog.Service{
		"shop/api": {Environments: map[string]catalog.ServiceEnvironment{"dev": {Peers: []catalog.Peer{}, Strict: true}}},
		"shop/web": {Environments: map[string]catalog.ServiceEnvironment{"dev": {Peers: []catalog.Peer{prober}}}},
	}}

	below := estate{"shop": {"dev": "identity"}}
	enforced := estate{"shop": {"dev": "enforced"}}

	assert.NoError(t, c.ValidateEstate(below), "strict is inert below enforced")

	err := c.ValidateEstate(enforced)
	require.Error(t, err)
	assert.ErrorContains(t, err, "shop/shop-prober")
	assert.ErrorContains(t, err, "no grant on shop/api")

	c.Services["shop/api"] = catalog.Service{Environments: map[string]catalog.ServiceEnvironment{
		"dev": {Peers: []catalog.Peer{prober}, Strict: true},
	}}
	assert.NoError(t, c.ValidateEstate(enforced))
	assert.Equal(t, []string{"api"}, c.StrictComponents("shop", "dev"))
	assert.Empty(t, c.StrictComponents("shop", "prod"))
}
