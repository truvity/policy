package chartschema_test

import (
	"encoding/json"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/truvity/policy/chartschema"
	"github.com/truvity/policy/config"
)

const service = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://example.com/echo.json",
  "allOf": [{"$ref": "https://github.com/truvity/policy/schemas/service.json"}],
  "type": "object",
  "unevaluatedProperties": false,
  "required": ["greeting"],
  "properties": {
    "listen": {"$ref": "https://github.com/truvity/policy/schemas/fragments/listen.json"},
    "greeting": {"type": "string", "minLength": 1}
  }
}`

const source = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "platform": {"$ref": "https://github.com/truvity/policy/schemas/fragments/platform.json"},
    "config": {"$ref": "echo.json"}
  }
}`

func fixture() fstest.MapFS {
	return fstest.MapFS{
		"chart/values.schema.src.json": {Data: []byte(source)},
		"chart/echo.json":              {Data: []byte(service)},
	}
}

func compose(t *testing.T, fsys fstest.MapFS) []byte {
	t.Helper()

	out, err := chartschema.Compose(fsys, "chart/values.schema.src.json")
	if err != nil {
		t.Fatal(err)
	}

	return out
}

// The composed schema must be usable with nothing around it: Helm validates
// offline, so a reference left pointing at a URL is a fetch it would make.
func TestComposedSchemaRefersToNothingOutsideItself(t *testing.T) {
	var doc map[string]any
	if err := json.Unmarshal(compose(t, fixture()), &doc); err != nil {
		t.Fatal(err)
	}

	defs, _ := doc["$defs"].(map[string]any)
	for _, id := range []string{
		"https://example.com/echo.json",
		"https://github.com/truvity/policy/schemas/service.json",
		"https://github.com/truvity/policy/schemas/fragments/listen.json",
		"https://github.com/truvity/policy/schemas/fragments/platform.json",
	} {
		if _, ok := defs[id]; !ok {
			t.Errorf("$defs does not embed %s; it has %d documents", id, len(defs))
		}
	}

	var walk func(v any)
	walk = func(v any) {
		switch n := v.(type) {
		case map[string]any:
			if ref, ok := n["$ref"].(string); ok && !strings.HasPrefix(ref, "#") {
				id, _, _ := strings.Cut(ref, "#")
				if _, ok := defs[id]; !ok {
					t.Errorf("$ref %q names a document that is not embedded", ref)
				}
			}
			for _, x := range n {
				walk(x)
			}
		case []any:
			for _, x := range n {
				walk(x)
			}
		}
	}
	walk(doc)
}

func TestComposeIsDeterministic(t *testing.T) {
	first, second := compose(t, fixture()), compose(t, fixture())
	if string(first) != string(second) {
		t.Error("the same inputs composed to different bytes: a committed schema could never be compared with it")
	}
	if !strings.HasSuffix(string(first), "}\n") {
		t.Error("the composed schema must end with a newline, like every file in this repository")
	}
}

// The composed schema must accept and refuse what its parts do: the service's
// own rules and the platform's, through one document.
func TestComposedSchemaValidatesBothBlocks(t *testing.T) {
	schema := compose(t, fixture())

	good := map[string]any{
		"platform": map[string]any{"replicas": 2},
		"config": map[string]any{
			"probes":   map[string]any{"address": ":7070"},
			"listen":   map[string]any{"address": ":8080"},
			"greeting": "hello",
		},
	}
	if err := config.Validate(good, schema); err != nil {
		t.Errorf("a valid values document was refused: %v", err)
	}

	for name, mutate := range map[string]func(v map[string]any){
		"an unknown platform key": func(v map[string]any) { v["platform"].(map[string]any)["replcas"] = 2 },
		"a platform value out of range": func(v map[string]any) {
			v["platform"].(map[string]any)["replicas"] = -1
		},
		"an unknown config key": func(v map[string]any) { v["config"].(map[string]any)["greting"] = "x" },
		"a config key the service requires, missing": func(v map[string]any) {
			delete(v["config"].(map[string]any), "greeting")
		},
		"a probes address the envelope refuses": func(v map[string]any) {
			v["config"].(map[string]any)["probes"] = map[string]any{"address": "7070"}
		},
	} {
		doc := map[string]any{}
		raw, _ := json.Marshal(good)
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		mutate(doc)
		if err := config.Validate(doc, schema); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

func TestComposeRefusesWhatHelmWouldHaveToFetch(t *testing.T) {
	fsys := fixture()
	fsys["chart/values.schema.src.json"] = &fstest.MapFile{Data: []byte(`{"properties": {"x": {"$ref": "https://example.org/elsewhere.json"}}}`)}
	if _, err := chartschema.Compose(fsys, "chart/values.schema.src.json"); err == nil {
		t.Error("a URL the composer cannot resolve was accepted")
	}

	fsys = fixture()
	fsys["chart/echo.json"] = &fstest.MapFile{Data: []byte(`{"type": "object"}`)}
	if _, err := chartschema.Compose(fsys, "chart/values.schema.src.json"); err == nil {
		t.Error("a referenced document with no $id was accepted: nothing can name it once embedded")
	}

	fsys = fixture()
	delete(fsys, "chart/echo.json")
	if _, err := chartschema.Compose(fsys, "chart/values.schema.src.json"); err == nil {
		t.Error("a reference to a file that does not exist was accepted")
	}
}
