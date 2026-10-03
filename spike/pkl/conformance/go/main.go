// Command pklconf runs every fixture in a manifest through the two Go-side
// validators of the spike and prints one JSON line per (fixture, validator):
//
//	jsonschema-hand  santhosh-tekuri/jsonschema on the hand-written schemas
//	jsonschema-gen   the same library on the schemas generated from Pkl
//	pkl-go           the Pkl contract loaded through pkl-go into the generated
//	                 Go structs (Pkl does the validating; the structs do none)
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/apple/pkl-go/pkl"
	"github.com/santhosh-tekuri/jsonschema/v6"
	yaml "go.yaml.in/yaml/v3"

	"spike.invalid/pklconf/gen/logarchiver"
	"spike.invalid/pklconf/gen/migrate"
	"spike.invalid/pklconf/gen/platform"
	"spike.invalid/pklconf/gen/prober"
	"spike.invalid/pklconf/gen/redirect"
	"spike.invalid/pklconf/gen/shortener"
	"spike.invalid/pklconf/gen/showcase"
	"spike.invalid/pklconf/gen/stat"
	"spike.invalid/pklconf/gen/urls"
	"spike.invalid/pklconf/gen/web"
)

type entry struct{ ID, Schema, Path string }
type doc struct {
	Gen, Pkl string
	Hand     *string
}

type result struct {
	ID        string `json:"id"`
	Validator string `json:"validator"`
	Accept    bool   `json:"accept"`
	Detail    string `json:"detail,omitempty"`
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func main() {
	spike, repo, genDir := os.Args[1], os.Args[2], os.Args[3] // spike/pkl dir, repo root, generated schema dir
	var manifest []entry
	raw, err := os.ReadFile(filepath.Join(spike, "conformance/manifest.json"))
	must(err)
	must(json.Unmarshal(raw, &manifest))
	var docs map[string]doc
	raw, err = os.ReadFile(filepath.Join(spike, "conformance/documents.json"))
	must(err)
	must(json.Unmarshal(raw, &docs))

	gen := loadSet(genDir, nil)
	var handPaths []string
	for _, d := range docs {
		if d.Hand != nil {
			handPaths = append(handPaths, filepath.Join(repo, *d.Hand))
		}
	}
	for _, g := range []string{"schemas/service.json", "schemas/fragments/*.json"} {
		m, _ := filepath.Glob(filepath.Join(repo, g))
		handPaths = append(handPaths, m...)
	}
	hand := loadSet("", handPaths)

	ctx := context.Background()
	ev, err := pkl.NewEvaluator(ctx, pkl.PreconfiguredOptions)
	must(err)
	defer ev.Close()

	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	emit := func(r result) { b, _ := json.Marshal(r); fmt.Fprintln(out, string(b)) }

	for _, e := range manifest {
		d := docs[e.Schema]
		instance := readYAML(e.Path)

		if d.Hand != nil {
			emit(validate(e.ID, "jsonschema-hand", hand, filepath.Join(repo, *d.Hand), instance))
		}
		emit(validate(e.ID, "jsonschema-gen", gen, filepath.Join(genDir, d.Gen), instance))
		emit(structJSON(e))
		for _, r := range loadPkl(ctx, ev, spike, e, d) {
			emit(r)
		}
	}
}

func readYAML(path string) any {
	b, err := os.ReadFile(path)
	must(err)
	var v any
	must(yaml.Unmarshal(b, &v))
	j, err := json.Marshal(v)
	must(err)
	inst, err := jsonschema.UnmarshalJSON(strings.NewReader(string(j)))
	must(err)
	return inst
}

// loadSet registers every document under the identifier it carries.
func loadSet(dir string, paths []string) map[string]any {
	if dir != "" {
		_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && filepath.Ext(p) == ".json" {
				paths = append(paths, p)
			}
			return nil
		})
	}
	set := map[string]any{}
	for _, p := range paths {
		f, err := os.Open(p)
		must(err)
		d, err := jsonschema.UnmarshalJSON(f)
		f.Close()
		must(err)
		set[p] = d
	}
	return set
}

func validate(id, name string, set map[string]any, root string, instance any) result {
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	for p, d := range set {
		m := d.(map[string]any)
		must(c.AddResource(m["$id"].(string), d))
		_ = p
	}
	sch, err := c.Compile(set[root].(map[string]any)["$id"].(string))
	if err != nil {
		return result{id, name, false, "schema: " + err.Error()}
	}
	if err := sch.Validate(instance); err != nil {
		return result{id, name, false, firstLine(err.Error())}
	}
	return result{id, name, true, ""}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i > 0 {
		s = s[:i]
	}
	return s
}

// structJSON decodes the fixture straight into the generated struct with
// encoding/json, without Pkl: what a service gets if it builds the generated
// type itself. The structs carry types and nothing else.
func structJSON(e entry) result {
	b, err := os.ReadFile(e.Path)
	must(err)
	var v any
	must(yaml.Unmarshal(b, &v))
	j, err := json.Marshal(v)
	must(err)
	var target any
	switch e.Schema {
	case "web":
		target = &web.WebImpl{}
	case "urls":
		target = &urls.UrlsImpl{}
	case "redirect":
		target = &redirect.RedirectImpl{}
	case "stat":
		target = &stat.StatImpl{}
	case "prober":
		target = &prober.ProberImpl{}
	case "migrate":
		target = &migrate.Migrate{}
	case "log":
		target = &logarchiver.LogArchiverImpl{}
	case "shortener":
		target = &shortener.ShortenerImpl{}
	case "platform":
		target = &platform.Platform{}
	case "showcase":
		target = &showcase.Showcase{}
	}
	if err := json.Unmarshal(j, target); err != nil {
		return result{e.ID, "go-struct", false, firstLine(err.Error())}
	}
	return result{e.ID, "go-struct", true, ""}
}

// loadPkl evaluates the fixture as a typed instance of the contract module and
// decodes it into the struct pkl-gen-go generated for that module.
func loadPkl(ctx context.Context, ev pkl.Evaluator, spike string, e entry, d doc) []result {
	src := fmt.Sprintf(`
import "pkl:yaml"
import "file://%s/gen/Load.pkl"
import "file://%s/%s" as M
local data = new yaml.Parser { useMapping = false }.parse(read("file://%s"))
output { value = Load.load(M, data) }
`, spike, spike, d.Pkl, e.Path)
	// pkl-eval: Pkl alone, no struct involved.
	if _, err := ev.EvaluateOutputText(ctx, pkl.TextSource(src)); err != nil {
		msg := firstLine(strings.ReplaceAll(err.Error(), "–– Pkl Error ––\n", ""))
		return []result{{e.ID, "pkl-eval", false, msg}, {e.ID, "pkl-go", false, msg}}
	}
	// pkl-go: the same evaluation, decoded into the generated struct.
	var err error
	switch e.Schema {
	case "web":
		var o web.Web
		err = ev.EvaluateOutputValue(ctx, pkl.TextSource(src), &o)
	case "urls":
		var o urls.Urls
		err = ev.EvaluateOutputValue(ctx, pkl.TextSource(src), &o)
	case "redirect":
		var o redirect.Redirect
		err = ev.EvaluateOutputValue(ctx, pkl.TextSource(src), &o)
	case "stat":
		var o stat.Stat
		err = ev.EvaluateOutputValue(ctx, pkl.TextSource(src), &o)
	case "prober":
		var o prober.Prober
		err = ev.EvaluateOutputValue(ctx, pkl.TextSource(src), &o)
	case "migrate":
		var o migrate.Migrate
		err = ev.EvaluateOutputValue(ctx, pkl.TextSource(src), &o)
	case "log":
		var o logarchiver.LogArchiver
		err = ev.EvaluateOutputValue(ctx, pkl.TextSource(src), &o)
	case "shortener":
		var o shortener.Shortener
		err = ev.EvaluateOutputValue(ctx, pkl.TextSource(src), &o)
	case "platform":
		var o platform.Platform
		err = ev.EvaluateOutputValue(ctx, pkl.TextSource(src), &o)
	case "showcase":
		var o showcase.Showcase
		err = ev.EvaluateOutputValue(ctx, pkl.TextSource(src), &o)
	default:
		return []result{{e.ID, "pkl-eval", true, ""}, {e.ID, "pkl-go", false, "no struct for " + e.Schema}}
	}
	if err != nil {
		return []result{{e.ID, "pkl-eval", true, ""}, {e.ID, "pkl-go", false, firstLine(err.Error())}}
	}
	return []result{{e.ID, "pkl-eval", true, ""}, {e.ID, "pkl-go", true, ""}}
}
