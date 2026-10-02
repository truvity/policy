package conformance_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/truvity/policy/conformance"
)

// A chart that follows the convention, as a render. The `echo` component
// listens on 8080, probes on 7070, and has a Service.
const goodRender = `apiVersion: v1
kind: ConfigMap
metadata:
  name: r-echo-config
data:
  echo.yaml: |
    listen:
      address: :8080
    probes:
      address: :7070
---
apiVersion: v1
kind: Service
metadata:
  name: r-echo
spec:
  ports:
    - name: http
      port: 8080
      targetPort: http
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: r-echo
spec:
  template:
    metadata:
      annotations:
        checksum/config: CHECKSUM
    spec:
      containers:
        - name: echo
          env:
            - name: OTEL_SERVICE_NAME
              value: r-echo
            - name: TOKEN
              valueFrom:
                secretKeyRef: {name: s, key: k}
          ports:
            - name: http
              containerPort: 8080
            - name: probes
              containerPort: 7070
          livenessProbe:
            httpGet: {path: /health/live, port: probes}
          readinessProbe:
            httpGet: {path: /health/ready, port: probes}
`

func echoConfig() map[string]any {
	return map[string]any{
		"listen": map[string]any{"address": ":8080"},
		"probes": map[string]any{"address": ":7070"},
	}
}

func TestConfigMapEqualsConfigAcceptsTheVerbatimFile(t *testing.T) {
	conformance.ConfigMapEqualsConfig(t, []byte(goodRender), "r-echo-config", "echo.yaml", echoConfig())
}

func TestConfigMapEqualsConfigCatchesAKeyAddedOrDropped(t *testing.T) {
	added := echoConfig()
	added["greeting"] = "hello"

	dropped := echoConfig()
	delete(dropped, "probes")

	for name, config := range map[string]map[string]any{"a key the file lacks": added, "a key the file has": dropped} {
		r := &recorder{TB: t}
		conformance.ConfigMapEqualsConfig(r, []byte(goodRender), "r-echo-config", "echo.yaml", config)
		if len(r.errs) == 0 {
			t.Errorf("%s: the difference was not reported", name)
		}
	}
}

func TestPortsEqualConfigAcceptsPortsDerivedFromTheFile(t *testing.T) {
	conformance.PortsEqualConfig(t, []byte(goodRender), "r-echo", echoConfig())
}

func TestPortsEqualConfigCatchesAPortThatIsNotWhatTheBinaryBinds(t *testing.T) {
	moved := echoConfig()
	moved["listen"] = map[string]any{"address": ":9090"}

	r := &recorder{TB: t}
	conformance.PortsEqualConfig(r, []byte(goodRender), "r-echo", moved)
	if len(r.errs) < 2 {
		t.Errorf("a listener moved in the file but not in the pod should be reported for the container and the Service, got %d: %v", len(r.errs), r.errs)
	}

	// Permissive transport adds a second port the render lacks.
	permissive := echoConfig()
	permissive["tls"] = map[string]any{"mode": "permissive", "address": ":8443"}
	r = &recorder{TB: t}
	conformance.PortsEqualConfig(r, []byte(goodRender), "r-echo", permissive)
	if len(r.errs) == 0 {
		t.Error("a permissive listener with no https port was not reported")
	}
}

func TestEnvIsDeclared(t *testing.T) {
	conformance.EnvIsDeclared(t, []byte(goodRender), "r-echo", []string{"TOKEN"})

	r := &recorder{TB: t}
	conformance.EnvIsDeclared(r, []byte(goodRender), "r-echo", nil)
	if len(r.errs) == 0 {
		t.Error("a secret that is not declared was not reported")
	}

	r = &recorder{TB: t}
	conformance.EnvIsDeclared(r, []byte(strings.Replace(goodRender, "OTEL_SERVICE_NAME", "GREETING", 1)), "r-echo", []string{"TOKEN"})
	if len(r.errs) == 0 {
		t.Error("an environment variable that carries configuration was not reported")
	}
}

func TestChartSchemaIsComposed(t *testing.T) {
	src := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {"platform": {"$ref": "https://github.com/truvity/policy/schemas/fragments/platform.json"}}
}`
	fsys := fstest.MapFS{"c/values.schema.src.json": {Data: []byte(src)}}

	// What is committed is whatever the first compose wrote: use it as the
	// "committed" file, then edit it.
	r := &recorder{TB: t}
	fsys["c/values.schema.json"] = &fstest.MapFile{Data: []byte("{}\n")}
	conformance.ChartSchemaIsComposed(r, fsys, "c/values.schema.src.json", "c/values.schema.json")
	if len(r.errs) == 0 {
		t.Error("a schema that is not the composed one was not reported")
	}
}
