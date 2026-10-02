// Package chartschema composes the values schema of a service chart.
//
// A service chart's values are two blocks and nothing else of its own
// (decision 0009): `platform`, the shape in schemas/fragments/platform.json,
// and `config`, which is exactly the service's own configuration schema. The
// chart's `values.schema.json` is therefore not written: it is COMPOSED from a
// small source file that says which service schema `config` is, so that the
// schema Helm validates the values with and the schema the binary validates
// its file with are the same document, and cannot be edited apart.
//
// Composing is bundling. Helm validates offline, and a `$ref` to a URL is a
// fetch it must not make, so every referenced document is embedded under
// `$defs`, keyed by its own `$id` (JSON Schema 2020-12's compound document),
// and each `$ref` is rewritten to that `$id`. Nothing in the result refers to
// anything outside it.
package chartschema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/truvity/policy"
)

// SourceName is the file a chart commits beside its `values.schema.json` to
// say what that schema is composed from.
const SourceName = "values.schema.src.json"

// Compose reads the source schema at src in fsys and returns the composed
// schema: the source with every `$ref` to another document resolved and that
// document embedded under `$defs` by its `$id`.
//
// A relative `$ref` is read from fsys, relative to the file that holds it, and
// the document it names must carry an absolute `$id`. An absolute one under
// [policy.SchemaBase] is read from the schemas this module embeds. Anything
// else is an error: a reference the composer cannot resolve is one Helm would
// have to fetch.
//
// The output is deterministic: the same inputs are the same bytes, which is
// what lets a test compare it with a committed file.
func Compose(fsys fs.FS, src string) ([]byte, error) {
	root, err := readDoc(fsys, src)
	if err != nil {
		return nil, err
	}

	c := &composer{fsys: fsys, defs: map[string]any{}}
	if err := c.walk(root, path.Dir(src), src); err != nil {
		return nil, err
	}

	if len(c.defs) > 0 {
		defs, _ := root["$defs"].(map[string]any)
		if defs == nil {
			defs = map[string]any{}
		}
		for id, doc := range c.defs {
			if _, clash := defs[id]; clash {
				return nil, fmt.Errorf("%s: $defs already has %q, which is an identifier the composer embeds by", src, id)
			}
			defs[id] = doc
		}
		root["$defs"] = defs
	}

	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(root); err != nil {
		return nil, fmt.Errorf("%s: %w", src, err)
	}

	return out.Bytes(), nil
}

type composer struct {
	fsys fs.FS
	defs map[string]any
}

func readDoc(fsys fs.FS, name string) (map[string]any, error) {
	raw, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, err
	}

	return parse(name, raw)
}

func parse(name string, raw []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	// Numbers stay as written: a `minimum: 1` must not become `1.0`.
	dec.UseNumber()

	var doc map[string]any
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("%s: not a JSON object: %w", name, err)
	}

	return doc, nil
}

// walk rewrites every `$ref` under node, embedding what each names. dir is
// where relative references in node are resolved from, and file names the
// document for an error.
func (c *composer) walk(node any, dir, file string) error {
	switch n := node.(type) {
	case map[string]any:
		if ref, ok := n["$ref"].(string); ok && !strings.HasPrefix(ref, "#") {
			id, err := c.embed(ref, dir, file)
			if err != nil {
				return err
			}
			n["$ref"] = id
		}
		for k, v := range n {
			if k == "$ref" {
				continue
			}
			if err := c.walk(v, dir, file); err != nil {
				return err
			}
		}
	case []any:
		for _, v := range n {
			if err := c.walk(v, dir, file); err != nil {
				return err
			}
		}
	}

	return nil
}

// embed makes sure the document ref names is in c.defs, and returns the
// reference to write in its place: the document's `$id`, with the fragment of
// ref, if any, kept.
func (c *composer) embed(ref, dir, file string) (string, error) {
	target, fragment, _ := strings.Cut(ref, "#")

	var (
		doc     map[string]any
		docDir  string
		docFile string
		err     error
	)
	switch {
	case strings.HasPrefix(target, policy.SchemaBase):
		docFile = "schemas/" + strings.TrimPrefix(target, policy.SchemaBase)
		var raw []byte
		raw, err = policy.Schemas.ReadFile(docFile)
		if err != nil {
			return "", fmt.Errorf("%s: $ref %q names no schema this module publishes", file, ref)
		}
		doc, err = parse(docFile, raw)
		if err != nil {
			return "", err
		}
		// Its own relative references (there are none today) would be
		// relative to the embedded tree, not to the chart.
		docDir = path.Dir(docFile)
	case strings.Contains(target, "://"):
		return "", fmt.Errorf("%s: $ref %q is a URL the composer cannot resolve: it embeds this module's schemas and files beside the source, and Helm must never fetch one", file, ref)
	default:
		docFile = path.Join(dir, target)
		doc, err = readDoc(c.fsys, docFile)
		if err != nil {
			return "", fmt.Errorf("%s: $ref %q: %w", file, ref, err)
		}
		docDir = path.Dir(docFile)
	}

	id, _ := doc["$id"].(string)
	if !strings.Contains(id, "://") {
		return "", fmt.Errorf("%s: has no absolute $id, so it cannot be embedded by one (referenced from %s)", docFile, file)
	}

	if _, seen := c.defs[id]; !seen {
		// A placeholder first, so a document that refers back to itself
		// terminates, then the real walk.
		c.defs[id] = doc
		delete(doc, "$schema")
		if err := c.walk(doc, docDir, docFile); err != nil {
			return "", err
		}
	}

	if fragment != "" {
		return id + "#" + fragment, nil
	}

	return id, nil
}
