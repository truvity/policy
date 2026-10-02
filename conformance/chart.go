package conformance

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	yaml "go.yaml.in/yaml/v3"

	"github.com/truvity/policy/chartschema"
)

// The checks in this file hold a SERVICE CHART to the library convention
// (docs/contracts/component.md, C15 to C17; decision 0009). Each takes what a
// chart rendered, or the files it committed, and compares them with what the
// chart's own `config` says — computed here, in Go, and never by asking the
// template, because a check that asks the thing it checks is not one.

// Manifests parses a rendered manifest stream into its documents.
func Manifests(t testing.TB, rendered []byte) []map[string]any {
	t.Helper()

	var docs []map[string]any
	dec := yaml.NewDecoder(bytes.NewReader(rendered))
	for {
		var d map[string]any
		err := dec.Decode(&d)
		if errors.Is(err, io.EOF) {
			return docs
		}
		if err != nil {
			t.Fatalf("the rendered manifests are not valid YAML: %v", err)
		}
		if d != nil {
			docs = append(docs, d)
		}
	}
}

func find(t testing.TB, docs []map[string]any, kind, name string) map[string]any {
	t.Helper()

	var names []string
	for _, d := range docs {
		if d["kind"] != kind {
			continue
		}
		got, _ := dig(d, "metadata", "name").(string)
		if got == name {
			return d
		}
		names = append(names, got)
	}
	t.Fatalf("the render has no %s named %q; found %v", kind, name, names)

	return nil
}

func dig(v any, keys ...string) any {
	for _, k := range keys {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[k]
	}

	return v
}

// asJSON normalises a document through JSON, so that YAML's integer, float
// and string spellings compare the way a loader would see them.
func asJSON(t testing.TB, v any) any {
	t.Helper()

	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("cannot represent %v as JSON: %v", v, err)
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}

	return out
}

// ConfigMapEqualsConfig fails the test unless the configuration file the
// render puts in the named ConfigMap, under key, is EXACTLY config — what the
// chart was given as `.Values.config`, with nothing added, dropped, renamed or
// defaulted on the way (C16).
//
// The comparison is of parsed documents, not of text: how the YAML is spelled
// is the renderer's business and the binary never sees it. What it does see —
// every key, every value, every list — is compared whole, so a field a
// template quietly set, or one it dropped, is a difference.
func ConfigMapEqualsConfig(t testing.TB, rendered []byte, configMap, key string, config map[string]any) {
	t.Helper()

	cm := find(t, Manifests(t, rendered), "ConfigMap", configMap)
	raw, ok := dig(cm, "data", key).(string)
	if !ok {
		t.Errorf("ConfigMap %s has no %q key", configMap, key)

		return
	}
	var got any
	if err := yaml.Unmarshal([]byte(raw), &got); err != nil {
		t.Errorf("ConfigMap %s's %s is not YAML: %v", configMap, key, err)

		return
	}

	if want := asJSON(t, config); !reflect.DeepEqual(asJSON(t, got), want) {
		t.Errorf("ConfigMap %s's %s is not the chart's `config`, verbatim:\n  rendered: %v\n  config:   %v\nThe binary reads the file, the schema validated the values; a chart that changes anything between them has two documents where the contract has one.", configMap, key, asJSON(t, got), want)
	}
}

// ChecksumFollowsConfig fails the test unless the Deployment's
// `checksum/config` pod annotation is the SHA-256 of the YAML the ConfigMap
// holds for key. Without the annotation a `helm upgrade` that only changes the
// file changes nothing the cluster sees: the pods keep the file they started
// with, and the deploy reads as successful.
func ChecksumFollowsConfig(t testing.TB, rendered []byte, deployment, configMap, key string) {
	t.Helper()

	docs := Manifests(t, rendered)
	d := find(t, docs, "Deployment", deployment)
	got, _ := dig(d, "spec", "template", "metadata", "annotations", "checksum/config").(string)
	raw, _ := dig(find(t, docs, "ConfigMap", configMap), "data", key).(string)

	sum := sha256.Sum256([]byte(strings.TrimSuffix(raw, "\n")))
	if want := hex.EncodeToString(sum[:]); got != want {
		t.Errorf("Deployment %s's checksum/config is %q, want %q (the checksum of ConfigMap %s's %s): the pods would not restart on a change to the file", deployment, got, want, configMap, key)
	}
}

// addressPort reads the port of a host:port address, the way the schema's own
// `pattern` defines one.
func addressPort(address string) (int, bool) {
	m := regexp.MustCompile(`:([0-9]{1,5})$`).FindStringSubmatch(address)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])

	return n, err == nil
}

// expectedPorts is what a process with this configuration binds, by the names
// the library gives them: `http` for `listen`, `probes` for `probes`, and
// `https` for `tls.address` when the transport is permissive (strict has one
// listener, the service's own, speaking TLS).
func expectedPorts(t testing.TB, config map[string]any) map[string]int {
	t.Helper()

	want := map[string]int{}
	add := func(name string, address any) {
		s, _ := address.(string)
		n, ok := addressPort(s)
		if !ok {
			t.Fatalf("config names %s as %q, which is not host:port", name, s)
		}
		want[name] = n
	}

	if listen, ok := config["listen"].(map[string]any); ok {
		add("http", listen["address"])
	}
	if probes, ok := config["probes"].(map[string]any); ok {
		add("probes", probes["address"])
	}
	if tls, ok := config["tls"].(map[string]any); ok && tls["mode"] == "permissive" {
		add("https", tls["address"])
	}

	return want
}

// PortsEqualConfig fails the test unless the ports the Deployment's container
// declares are exactly the ports the component's own `config` binds, and
// unless the Service (when the component listens) forwards exactly the
// listener's ports to them (C16: ports must always equal what the binary
// listens on).
//
// A port written beside the file is two numbers for one fact. The pod is then
// reachable on one while the process listens on the other, and nothing says
// so: the symptom is a probe that never passes, or a Service that resets every
// connection, in a cluster.
func PortsEqualConfig(t testing.TB, rendered []byte, workload string, config map[string]any) {
	t.Helper()

	docs := Manifests(t, rendered)
	d := find(t, docs, "Deployment", workload)
	want := expectedPorts(t, config)

	got := map[string]int{}
	containers, _ := dig(d, "spec", "template", "spec", "containers").([]any)
	if len(containers) == 0 {
		t.Fatalf("Deployment %s has no container", workload)
	}
	ports, _ := containers[0].(map[string]any)["ports"].([]any)
	for _, p := range ports {
		pm, _ := p.(map[string]any)
		name, _ := pm["name"].(string)
		port, _ := pm["containerPort"].(int)
		got[name] = port
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Deployment %s declares container ports %v, but its config binds %v", workload, sortedPorts(got), sortedPorts(want))
	}

	// The probes are on the named port, so they follow the listener.
	for _, kind := range []string{"livenessProbe", "readinessProbe"} {
		if port := dig(containers[0], kind, "httpGet", "port"); port != "probes" {
			t.Errorf("Deployment %s's %s is on port %v, not the `probes` port its config binds", workload, kind, port)
		}
	}

	_, listens := want["http"]
	var svc map[string]any
	for _, s := range docs {
		if name, _ := dig(s, "metadata", "name").(string); s["kind"] == "Service" && name == workload {
			svc = s
		}
	}
	switch {
	case !listens && svc != nil:
		t.Errorf("a Service %s exists, but the component's config has no `listen`", workload)
	case listens && svc == nil:
		t.Errorf("the component's config has a `listen` and no Service %s forwards to it", workload)
	case listens:
		svcPorts := map[string]int{}
		targets := map[string]string{}
		items, _ := dig(svc, "spec", "ports").([]any)
		for _, p := range items {
			pm, _ := p.(map[string]any)
			name, _ := pm["name"].(string)
			svcPorts[name], _ = pm["port"].(int)
			targets[name], _ = pm["targetPort"].(string)
		}
		wantSvc := map[string]int{"http": want["http"]}
		if n, ok := want["https"]; ok {
			wantSvc["https"] = n
		}
		if !reflect.DeepEqual(svcPorts, wantSvc) {
			t.Errorf("Service %s forwards %v, but the listener binds %v (the probes listener is never in a Service)", workload, sortedPorts(svcPorts), sortedPorts(wantSvc))
		}
		for name, target := range targets {
			if target != name {
				t.Errorf("Service %s sends %s to %q, not the container port of the same name", workload, name, target)
			}
		}
	}
}

func sortedPorts(m map[string]int) string {
	var parts []string
	for k, v := range m {
		parts = append(parts, fmt.Sprintf("%s=%d", k, v))
	}
	sort.Strings(parts)

	return "[" + strings.Join(parts, " ") + "]"
}

// EnvIsDeclared fails the test unless every environment variable the
// Deployment's container has is accounted for: an OpenTelemetry variable
// (decision 0006), one of the declared secrets (the keys of `platform.secrets`,
// decision 0002), or one the caller names in allowed (the environment a
// platform client library reads). A variable that is none of those is a second
// way to configure the service, which is what 0002 refuses.
func EnvIsDeclared(t testing.TB, rendered []byte, deployment string, secrets []string, allowed ...string) {
	t.Helper()

	d := find(t, Manifests(t, rendered), "Deployment", deployment)
	containers, _ := dig(d, "spec", "template", "spec", "containers").([]any)
	if len(containers) == 0 {
		t.Fatalf("Deployment %s has no container", deployment)
	}
	env, _ := containers[0].(map[string]any)["env"].([]any)

	declared := map[string]bool{}
	for _, s := range secrets {
		declared[s] = true
	}
	ok := map[string]bool{}
	for _, a := range allowed {
		ok[a] = true
	}

	seen := map[string]bool{}
	for _, e := range env {
		em, _ := e.(map[string]any)
		name, _ := em["name"].(string)
		seen[name] = true
		_, fromSecret := dig(em, "valueFrom", "secretKeyRef").(map[string]any)

		switch {
		case strings.HasPrefix(name, "OTEL_"):
		case declared[name]:
			if !fromSecret {
				t.Errorf("Deployment %s: %s is a declared secret and is not read from a Secret", deployment, name)
			}
		case fromSecret:
			t.Errorf("Deployment %s: %s is read from a Secret and is not declared in platform.secrets", deployment, name)
		case ok[name]:
		default:
			t.Errorf("Deployment %s has an environment variable %s that is neither telemetry, a declared secret nor one the chart allows: the file is the only structural input (decision 0002)", deployment, name)
		}
	}
	for s := range declared {
		if !seen[s] {
			t.Errorf("Deployment %s does not carry the declared secret %s", deployment, s)
		}
	}
}

// ChartSchemaIsComposed fails the test unless the committed values schema is
// byte for byte what its source composes to (C17): the platform schema, and
// `config` as the service's own schema, bundled by package chartschema. A
// schema edited by hand is a chart whose values and whose binary validate
// against two documents that were once one.
//
// src is the path of the chart's values.schema.src.json in fsys; committed is
// the path of the values.schema.json beside it.
func ChartSchemaIsComposed(t testing.TB, fsys fs.FS, src, committed string) {
	t.Helper()

	composed, err := chartschema.Compose(fsys, src)
	if err != nil {
		t.Errorf("%s does not compose: %v", src, err)

		return
	}
	have, err := fs.ReadFile(fsys, committed)
	if err != nil {
		t.Errorf("%v: run `just chart-schemas`", err)

		return
	}
	if !bytes.Equal(have, composed) {
		t.Errorf("%s is not what %s composes to: it was edited by hand, or the service schema moved. Run `just chart-schemas` and commit the result", committed, src)
	}
}
