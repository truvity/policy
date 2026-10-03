// Command compose builds, with the repository's own chartschema package, the
// values schema each chart would have if its `config` were the hand-written
// schema, so the schema generated from Pkl can be compared with it; and it
// validates each generated values.yaml against both, the way Helm does
// (Helm validates with the same library).
//
//	go run ./spike/pkl/conformance/compose <repo> <generated helm dir> <hand-composed output dir>
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing/fstest"

	"github.com/santhosh-tekuri/jsonschema/v6"
	yaml "go.yaml.in/yaml/v3"

	"github.com/truvity/policy/chartschema"
)

const src = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "%[1]s chart values",
  "description": "Two blocks and the image map: ` + "`platform`" + ` is the platform schema, ` + "`config`" + ` is the service's own schema, and the chart adds nothing between them.",
  "type": "object",
  "additionalProperties": false,
  "required": ["config", "images"],
  "properties": {
    "service-lib": {
      "type": "object",
      "additionalProperties": false,
      "properties": { "global": { "type": "object" } },
      "description": "Not a value: Helm puts a key named for every sub-chart into the values it validates."
    },
    "platform": { "$ref": "https://github.com/truvity/policy/schemas/fragments/platform.json" },
    "config": { "$ref": "%[1]s.schema.json" },
    "images": {
      "type": "object",
      "additionalProperties": false,
      "required": ["%[1]s"],
      "properties": {
        "%[1]s": { "$ref": "https://github.com/truvity/policy/schemas/fragments/platform.json#/properties/image" }
      }
    }
  }
}`

var component = map[string]string{
	"web": "examples/url-shortener/schemas/web.json", "urls": "examples/url-shortener/schemas/urls.json",
	"redirect": "examples/url-shortener/schemas/redirect.json", "stat": "examples/url-shortener/schemas/stat.json",
	"prober": "examples/url-shortener/schemas/prober.json", "migrate": "examples/url-shortener/schemas/migrate.json",
	"log": "examples/url-shortener/log/src/url_shortener_log/log.schema.json",
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	repo, genDir, handOut := os.Args[1], os.Args[2], os.Args[3]
	for name, path := range component {
		b, err := os.ReadFile(filepath.Join(repo, path))
		must(err)
		fsys := fstest.MapFS{
			"values.schema.src.json": {Data: []byte(fmt.Sprintf(src, name))},
			name + ".schema.json":    {Data: b},
		}
		composed, err := chartschema.Compose(fsys, "values.schema.src.json")
		must(err)
		must(os.MkdirAll(filepath.Join(handOut, name), 0o755))
		must(os.WriteFile(filepath.Join(handOut, name, "values.schema.json"), composed, 0o644))

		vals, err := os.ReadFile(filepath.Join(genDir, name, "values.yaml"))
		must(err)
		var v any
		must(yaml.Unmarshal(vals, &v))
		j, _ := json.Marshal(v)
		inst, err := jsonschema.UnmarshalJSON(strings.NewReader(string(j)))
		must(err)
		for label, file := range map[string]string{"hand-composed": filepath.Join(handOut, name, "values.schema.json"), "generated": filepath.Join(genDir, name, "values.schema.json")} {
			f, err := os.Open(file)
			must(err)
			doc, err := jsonschema.UnmarshalJSON(f)
			f.Close()
			must(err)
			c := jsonschema.NewCompiler()
			c.DefaultDraft(jsonschema.Draft2020)
			must(c.AddResource("mem://values.schema.json", doc))
			sch, err := c.Compile("mem://values.schema.json")
			if err != nil {
				fmt.Printf("%-9s generated values.yaml vs %-13s schema: SCHEMA ERROR %v\n", name, label, err)
				continue
			}
			if err := sch.Validate(inst); err != nil {
				fmt.Printf("%-9s generated values.yaml vs %-13s schema: REJECT %.120s\n", name, label, strings.ReplaceAll(err.Error(), "\n", " "))
			} else {
				fmt.Printf("%-9s generated values.yaml vs %-13s schema: accept\n", name, label)
			}
		}
	}
}
