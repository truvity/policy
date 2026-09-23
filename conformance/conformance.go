// Package conformance holds the test helpers that make the configuration
// contract checkable, so that a service and the chart that deploys it are
// held to one schema rather than to two beliefs about it.
//
// Import it from tests only.
package conformance

import (
	"errors"
	"io"
	"reflect"
	"sort"
	"strings"
	"testing"

	yaml "go.yaml.in/yaml/v3"

	"github.com/truvity/policy"
	"github.com/truvity/policy/config"
)

// TypeMatchesSchema fails the test when the configuration type and the schema
// disagree about which fields exist.
//
// This is the drift check the contract asks for. It deliberately compares the
// SET OF FIELDS rather than regenerating a schema and diffing it: a generated
// schema differs from a hand-written one in a dozen cosmetic ways — where
// definitions live, how nullability is spelled, which annotations survive —
// and a test that fails on all of them is a test people regenerate rather
// than read.
//
// What it catches is the failure that actually happens: a field renamed on
// one side only. That is worth a test; the cosmetics are not.
func TypeMatchesSchema(t testing.TB, v any, schema []byte) {
	t.Helper()

	inSchema := map[string]bool{}
	opaque := map[string]bool{}
	collectSchemaProperties(mustParse(t, schema), "", inSchema, opaque)

	inType := map[string]bool{}
	collectTypeFields(reflect.TypeOf(v), "", inType)

	var onlyType, onlySchema []string
	for k := range inType {
		if !inSchema[k] && !under(opaque, k) {
			onlyType = append(onlyType, k)
		}
	}
	for k := range inSchema {
		if !inType[k] {
			onlySchema = append(onlySchema, k)
		}
	}
	sort.Strings(onlyType)
	sort.Strings(onlySchema)

	for _, k := range onlyType {
		t.Errorf("%T has %s; the schema does not. A field the schema does not describe cannot be validated, so nothing stops a deployment from omitting it.", v, k)
	}
	for _, k := range onlySchema {
		t.Errorf("the schema has %s; %T does not. A key the service cannot read is a key a deployment can set and watch do nothing.", k, v)
	}
}

// under reports whether any ancestor of path is opaque — a subtree the schema
// describes through a reference this package cannot resolve, where comparing
// fields would report drift that is not there.
func under(opaque map[string]bool, path string) bool {
	for p := path; p != ""; {
		i := strings.LastIndex(p, ".")
		if i < 0 {
			break
		}
		p = p[:i]
		if opaque[p] {
			return true
		}
	}
	return false
}

// ValidDocument fails the test when doc does not satisfy schema. A chart's
// tests call it with what they render, using the same schema the binary
// validates with at start-up, which is what stops the two drifting.
func ValidDocument(t testing.TB, doc []byte, schema []byte) {
	t.Helper()
	var parsed any
	if err := yaml.Unmarshal(doc, &parsed); err != nil {
		t.Fatalf("the rendered document is not valid YAML: %v", err)
	}
	// Through JSON, exactly as the loader does, so that this test and the
	// running service see the same document.
	if err := config.Validate(roundTrip(t, parsed), schema); err != nil {
		t.Fatalf("the rendered configuration would be refused at start-up:\n%v", err)
	}
}

// ConfigMapData returns the value of key from the first ConfigMap in a
// rendered manifest stream that has it.
//
// A chart renders many documents; the configuration is one string inside one
// of them, and a test that reaches it by index breaks the first time a
// template is added.
func ConfigMapData(t testing.TB, rendered []byte, key string) []byte {
	t.Helper()
	dec := yaml.NewDecoder(strings.NewReader(string(rendered)))
	var names []string
	for {
		var m struct {
			Kind     string                `yaml:"kind"`
			Metadata struct{ Name string } `yaml:"metadata"`
			Data     map[string]string     `yaml:"data"`
		}
		err := dec.Decode(&m)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("the rendered manifests are not valid YAML: %v", err)
		}
		if m.Kind != "ConfigMap" {
			continue
		}
		names = append(names, m.Metadata.Name)
		if v, ok := m.Data[key]; ok {
			return []byte(v)
		}
	}
	t.Fatalf("no ConfigMap in the render has a %q key; found %v", key, names)
	return nil
}

func mustParse(t testing.TB, schema []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := yaml.Unmarshal(schema, &m); err != nil {
		t.Fatalf("the schema is not valid JSON: %v", err)
	}
	return m
}

func roundTrip(t testing.TB, v any) any {
	t.Helper()
	b, err := yaml.Marshal(v)
	if err != nil {
		t.Fatalf("cannot re-encode the document: %v", err)
	}
	var out any
	if err := yaml.Unmarshal(b, &out); err != nil {
		t.Fatalf("cannot re-read the document: %v", err)
	}
	return out
}

// collectSchemaProperties walks `properties`, through the combinators a
// strict schema uses to compose fragments, and through references to the
// shapes this repository publishes — which are resolved from the embedded
// copies, because a fragment's fields are part of the configuration exactly
// as the service's own are.
//
// A reference it cannot resolve marks that path opaque rather than pretending
// the subtree is empty: reporting every field under someone else's schema as
// drift is how a useful check becomes one people disable.
func collectSchemaProperties(schema map[string]any, prefix string, out, opaque map[string]bool) {
	if ref, ok := schema["$ref"].(string); ok {
		if resolved := resolveShared(ref); resolved != nil {
			collectSchemaProperties(resolved, prefix, out, opaque)
		} else if prefix != "" {
			opaque[prefix] = true
		}
	}
	if props, ok := schema["properties"].(map[string]any); ok {
		for name, sub := range props {
			p := name
			if prefix != "" {
				p = prefix + "." + name
			}
			out[p] = true
			if m, ok := sub.(map[string]any); ok {
				collectSchemaProperties(m, p, out, opaque)
			}
		}
	}
	for _, key := range []string{"allOf", "anyOf", "oneOf"} {
		if list, ok := schema[key].([]any); ok {
			for _, item := range list {
				if m, ok := item.(map[string]any); ok {
					collectSchemaProperties(m, prefix, out, opaque)
				}
			}
		}
	}
	if items, ok := schema["items"].(map[string]any); ok {
		collectSchemaProperties(items, prefix+"[]", out, opaque)
	}
}

// resolveShared returns the embedded schema a `$ref` names, or nil when the
// reference is to something this package does not carry.
func resolveShared(ref string) map[string]any {
	if !strings.HasPrefix(ref, policy.SchemaBase) {
		return nil
	}
	b, err := policy.Schemas.ReadFile("schemas/" + strings.TrimPrefix(ref, policy.SchemaBase))
	if err != nil {
		return nil
	}
	var m map[string]any
	if err := yaml.Unmarshal(b, &m); err != nil {
		return nil
	}
	return m
}

func collectTypeFields(t reflect.Type, prefix string, out map[string]bool) {
	for t != nil && (t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice) {
		if t.Kind() == reflect.Slice {
			prefix += "[]"
		}
		t = t.Elem()
	}
	if t == nil || t.Kind() != reflect.Struct {
		return
	}
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "-" {
			continue
		}
		if name == "" {
			name = f.Name
		}
		p := name
		if prefix != "" {
			p = prefix + "." + name
		}
		out[p] = true
		collectTypeFields(f.Type, p, out)
	}
}
