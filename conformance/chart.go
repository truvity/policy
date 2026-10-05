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

// podSpec finds the named workload (a Deployment, StatefulSet, DaemonSet, Job
// or CronJob) and returns its pod spec.
func podSpec(t testing.TB, rendered []byte, name string) map[string]any {
	t.Helper()

	for _, d := range Manifests(t, rendered) {
		got, _ := dig(d, "metadata", "name").(string)
		if got != name {
			continue
		}
		var spec any
		switch d["kind"] {
		case "Deployment", "StatefulSet", "DaemonSet", "Job":
			spec = dig(d, "spec", "template", "spec")
		case "CronJob":
			spec = dig(d, "spec", "jobTemplate", "spec", "template", "spec")
		default:
			continue
		}
		if m, ok := spec.(map[string]any); ok {
			return m
		}
	}
	t.Fatalf("the render has no workload named %q", name)

	return nil
}

// containersOf returns the containers and initContainers of a pod spec.
func containersOf(spec map[string]any) []map[string]any {
	var out []map[string]any
	for _, k := range []string{"initContainers", "containers"} {
		list, _ := spec[k].([]any)
		for _, c := range list {
			if m, ok := c.(map[string]any); ok {
				out = append(out, m)
			}
		}
	}

	return out
}

// EnvIsDeclared fails the test unless every environment variable of every
// container and init container of the workload `deployment` (a Deployment,
// StatefulSet, DaemonSet, Job or CronJob) is accounted for: an OpenTelemetry
// variable (decision 0006) or one the caller names in allowed (the environment a
// platform client library reads). A variable that is neither is a second way to
// configure the service, which is what 0002 refuses.
//
// A secret is never an environment variable (decision 0012): a variable read from
// a Secret (`valueFrom.secretKeyRef`) fails whatever its name, so does one whose
// name is in allowed, and so does any `envFrom` that names a Secret. Secrets are
// files, see [SecretsAreFiles].
func EnvIsDeclared(t testing.TB, rendered []byte, deployment string, allowed ...string) {
	t.Helper()

	spec := podSpec(t, rendered, deployment)
	containers := containersOf(spec)
	if len(containers) == 0 {
		t.Fatalf("%s has no container", deployment)
	}
	ok := map[string]bool{}
	for _, a := range allowed {
		ok[a] = true
	}

	for _, c := range containers {
		cname, _ := c["name"].(string)
		from, _ := c["envFrom"].([]any)
		for _, f := range from {
			if _, isSecret := dig(f, "secretRef").(map[string]any); isSecret {
				t.Errorf("%s, container %s: envFrom reads a Secret into the environment: a secret is a file under the configuration's secrets.root (decision 0012)", deployment, cname)
			}
		}
		env, _ := c["env"].([]any)
		for _, e := range env {
			em, _ := e.(map[string]any)
			name, _ := em["name"].(string)
			_, fromSecret := dig(em, "valueFrom", "secretKeyRef").(map[string]any)

			switch {
			case fromSecret:
				t.Errorf("%s, container %s: %s is read from a Secret into the environment: a secret is a file under the configuration's secrets.root, never a variable (decision 0012)", deployment, cname, name)
			case strings.HasPrefix(name, "OTEL_"):
			case ok[name]:
			default:
				t.Errorf("%s, container %s has an environment variable %s that is neither telemetry nor one the chart allows: the file is the only structural input (decision 0002)", deployment, cname, name)
			}
		}
	}
}

// fileMode reads a Kubernetes file mode as the API server would: a YAML integer
// (the decoder reads 0440 as octal and 288 as decimal, which are the same
// number) or, defensively, a string of octal digits.
func fileMode(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int64:
		return int(x), true
	case uint64:
		return int(x), true
	case float64:
		return int(x), true
	case string:
		n, err := strconv.ParseInt(strings.TrimPrefix(x, "0o"), 8, 32)
		return int(n), err == nil
	}

	return 0, false
}

// SecretsAreFiles fails the test unless every Secret the workload projects is a
// file nobody but the owner and the group can read, and no one can write: every
// volume with a `secret` or a projected `secret` source has a `defaultMode` of
// 0440 or tighter (an absent one is 0644) and no item `mode` looser, and every
// container that mounts it does so read-only. A workload with no such volume
// passes.
func SecretsAreFiles(t testing.TB, rendered []byte, deployment string) {
	t.Helper()

	spec := podSpec(t, rendered, deployment)
	volumes, _ := spec["volumes"].([]any)
	secretVolumes := map[string]bool{}
	for _, v := range volumes {
		vm, _ := v.(map[string]any)
		name, _ := vm["name"].(string)

		var secrets []map[string]any
		var defaultMode any
		if s, ok := vm["secret"].(map[string]any); ok {
			secrets, defaultMode = append(secrets, s), s["defaultMode"]
		}
		if p, ok := vm["projected"].(map[string]any); ok {
			sources, _ := p["sources"].([]any)
			for _, src := range sources {
				if s, ok := dig(src, "secret").(map[string]any); ok {
					secrets = append(secrets, s)
				}
			}
			if len(secrets) > 0 {
				defaultMode = p["defaultMode"]
			}
		}
		if len(secrets) == 0 {
			continue
		}
		secretVolumes[name] = true

		const tightest = 0o440
		mode, ok := fileMode(defaultMode)
		if !ok {
			t.Errorf("%s: volume %s projects a Secret with no defaultMode, which is 0644; want 0440 or tighter", deployment, name)
		} else if mode&^tightest != 0 {
			t.Errorf("%s: volume %s has defaultMode %#o, want 0440 or tighter", deployment, name, mode)
		}
		for _, s := range secrets {
			items, _ := s["items"].([]any)
			for _, it := range items {
				m := dig(it, "mode")
				if m == nil {
					continue
				}
				if im, ok := fileMode(m); !ok || im&^tightest != 0 {
					t.Errorf("%s: volume %s has an item mode looser than 0440", deployment, name)
				}
			}
		}
	}

	for _, c := range containersOf(spec) {
		mounts, _ := c["volumeMounts"].([]any)
		for _, m := range mounts {
			mm, _ := m.(map[string]any)
			if n, _ := mm["name"].(string); secretVolumes[n] && mm["readOnly"] != true {
				t.Errorf("%s: container %v mounts the secrets volume %s read-write", deployment, c["name"], n)
			}
		}
	}
}

// envSpelling is a key that says a value comes from a variable: `…Env`,
// `…EnvVar` and `…FromEnv`, in any case.
var envSpelling = regexp.MustCompile(`(?i).(env|envvar|fromenv)$`)

// NoEnvSecretFields fails the test when a configuration document (every YAML
// document in doc) has a key ending `Env`, `EnvVar` or `FromEnv`, in any case,
// at any depth (config.md rule 5, decision 0012). `…Env` said how a secret
// arrived and not which secret it was; a secret is a `…Secret` field holding a
// NAME under the service's one `secrets` source.
//
// Run it on the document of every version a binary reads from v2 on; a v1
// document that still has them is the one the binary's upgrade converts. allowed
// names the dotted paths of a key of that form that is not a secret, which is
// rare enough that each is written down.
func NoEnvSecretFields(t testing.TB, doc []byte, allowed ...string) {
	t.Helper()

	ok := map[string]bool{}
	for _, a := range allowed {
		ok[a] = true
	}
	var walk func(v any, path string)
	walk = func(v any, path string) {
		switch x := v.(type) {
		case map[string]any:
			for k, c := range x {
				p := k
				if path != "" {
					p = path + "." + k
				}
				if envSpelling.MatchString(k) && !ok[p] {
					t.Errorf("%s names a variable: a secret is a NAME in a field ending Secret, resolved through the one secrets source (config.md rule 5)", p)
				}
				walk(c, p)
			}
		case []any:
			for i, c := range x {
				walk(c, fmt.Sprintf("%s[%d]", path, i))
			}
		}
	}

	dec := yaml.NewDecoder(bytes.NewReader(doc))
	for {
		var root any
		err := dec.Decode(&root)
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			t.Fatalf("the document is not valid YAML: %v", err)
		}
		walk(root, "")
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
