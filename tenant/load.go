package tenant

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"regexp"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/truvity/policy"
	"github.com/truvity/policy/config"
)

// section is one file of a bundle and the schema that holds its shape.
type section struct {
	file   string
	schema string
	// decode reads the (already schema-valid) file into the bundle.
	decode func(dec *yaml.Decoder, b *Bundle) error
}

// wrap decodes a file whose rows sit under one top-level key.
func wrap[T any](assign func(*Bundle, T)) func(*yaml.Decoder, *Bundle) error {
	return func(dec *yaml.Decoder, b *Bundle) error {
		var v T
		if err := dec.Decode(&v); err != nil && !errors.Is(err, io.EOF) {
			return err
		}

		assign(b, v)

		return nil
	}
}

// sections is every file a bundle may hold, in the order refusals are
// reported. The manifest is first: it says which interface the rest is
// written for.
var sections = []section{
	{"tenant.yaml", "manifest.json", wrap(func(b *Bundle, v Manifest) { b.Manifest = v })},
	{"projects.yaml", "projects.json", wrap(func(b *Bundle, v struct {
		Projects []Project `yaml:"projects"`
	}) {
		b.Projects = v.Projects
	})},
	{"clients.yaml", "clients.json", wrap(func(b *Bundle, v struct {
		Clients []Client `yaml:"clients"`
	}) {
		b.Clients = v.Clients
	})},
	{"access.yaml", "access.json", wrap(func(b *Bundle, v Access) { b.Access = v })},
	{"alerts.yaml", "alerts.json", wrap(func(b *Bundle, v Alerts) { b.Alerts = v })},
	{"status.yaml", "status.json", wrap(func(b *Bundle, v struct {
		Entities []StatusEntity `yaml:"entities"`
	}) {
		b.Status = v.Entities
	})},
	{"ci.yaml", "ci.json", wrap(func(b *Bundle, v struct {
		Requests []CIRequest `yaml:"requests"`
	}) {
		b.CI = v.Requests
	})},
	{"registry.yaml", "registry.json", wrap(func(b *Bundle, v struct {
		Projects []RegistryRow `yaml:"projects"`
	}) {
		b.Registry = v.Projects
	})},
	{"github.yaml", "github.json", wrap(func(b *Bundle, v GitHub) { b.GitHub = &v })},
}

const manifestFile = "tenant.yaml"

// Load reads a bundle from fsys, rooted at the bundle directory (the
// `tenant/` directory of a company's repository, or an unpacked artifact).
//
// Every file is checked against its schema under schemas/tenant.v1 and
// decoded strictly. A file the contract does not name is refused. The manifest
// is read first: a bundle written for an interface newer than [Interface], or
// for another bundle format, is refused alone, because nothing else in it can
// be read. Every other failure in every file is reported in one pass.
//
// The result of a successful Load is not yet checked: see [Bundle.Validate]
// and [Bundle.Check]. Every list in it is non-nil.
func Load(fsys fs.FS) (*Bundle, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("tenant: %w", err)
	}

	c := &collector{}
	present := map[string][]byte{}

	for _, e := range entries {
		sec := sectionFor(e.Name())
		if sec == nil || e.IsDir() {
			c.add(e.Name(), "", UnknownField, "not a file the tenant contract names")
			continue
		}

		if !e.Type().IsRegular() {
			c.add(e.Name(), "", Invalid, "not a regular file: a link or a device would make two checkouts of one commit differ")
			continue
		}

		data, err := fs.ReadFile(fsys, e.Name())
		if err != nil {
			return nil, fmt.Errorf("tenant: %w", err)
		}

		present[e.Name()] = data
	}

	// The manifest first.
	manifest, ok := present[manifestFile]
	if !ok {
		c.add(manifestFile, "", Invalid, "a bundle needs its manifest")
		return nil, c.result()
	}

	var doc any
	if err := yaml.Unmarshal(manifest, &doc); err != nil {
		c.add(manifestFile, "", Invalid, "not valid YAML")
		return nil, c.result()
	}

	if m, ok := doc.(map[string]any); ok {
		if company, ok := m["company"].(string); ok {
			c.company = company
		}

		if !checkInterface(c, m) {
			return nil, c.result()
		}
	}

	b := &Bundle{}

	for i := range sections {
		sec := &sections[i]

		data, ok := present[sec.file]
		if !ok {
			continue
		}

		loadSection(c, sec, data, b)
	}

	if len(c.errs) > 0 {
		return nil, c.result()
	}

	b.normalize()

	return b, nil
}

func sectionFor(name string) *section {
	for i := range sections {
		if sections[i].file == name {
			return &sections[i]
		}
	}

	return nil
}

// checkInterface refuses a manifest written for an interface this package
// does not read. It reports false when the rest of the bundle cannot be read.
func checkInterface(c *collector, m map[string]any) bool {
	ok := true

	if s, present := m["schema"]; present && s != Schema {
		c.add(manifestFile, "schema", UnknownInterface, "not a bundle format this checker reads (it reads %s)", Schema)

		ok = false
	}

	if n, isInt := m["interface"].(int); isInt && n > Interface {
		c.add(manifestFile, "interface", UnknownInterface,
			"interface %d is newer than this checker reads (%d to %d)", n, OldestInterface, Interface)

		ok = false
	}

	return ok
}

// loadSection checks one file against its schema and, if it holds, decodes it.
func loadSection(c *collector, sec *section, data []byte, b *Bundle) {
	doc, err := oneDocument(data)
	if err != nil {
		c.add(sec.file, "", Invalid, "%s", err.Error())
		return
	}

	// An empty file is a section with no rows.
	if doc == nil {
		doc = map[string]any{}
		data = []byte("{}")
	}

	schema, err := policy.Schemas.ReadFile("schemas/tenant.v1/" + sec.schema)
	if err != nil {
		c.add(sec.file, "", Invalid, "no schema for this file: %v", err)
		return
	}

	if err := config.Validate(doc, schema); err != nil {
		var ce *config.Error
		if !errors.As(err, &ce) || len(ce.Failures) == 0 {
			c.add(sec.file, "", Invalid, "cannot be checked: %v", err)
			return
		}

		for _, f := range ce.Failures {
			path, msg := splitFailure(f)

			if keys, ok := unknownFields(msg); ok {
				// The schema reports the object; the error names the field,
				// so that a person reading it is not left to find it.
				for _, k := range keys {
					c.add(sec.file, join(path, k), UnknownField, "not a field the tenant contract names")
				}

				continue
			}

			c.add(sec.file, path, Invalid, "%s", msg)
		}

		return
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	if err := sec.decode(dec, b); err != nil {
		c.add(sec.file, "", Invalid, "cannot be read: %v", err)
		return
	}
}

// oneDocument reads the single YAML document of a file. A second document
// would be content the schema and the decoder never saw while the pinned
// bundle still carries it, so it is refused; an empty trailing document holds
// nothing and is not.
func oneDocument(data []byte) (any, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))

	var doc any
	if err := dec.Decode(&doc); err != nil && !errors.Is(err, io.EOF) {
		return nil, errors.New("not valid YAML")
	}

	var more any

	switch err := dec.Decode(&more); {
	case errors.Is(err, io.EOF):
		return doc, nil
	case err != nil:
		return nil, errors.New("not valid YAML")
	case more != nil:
		return nil, errors.New("more than one YAML document; a file holds one")
	}

	return doc, nil
}

// splitFailure turns config's "projects.0.name: message" into the path
// "projects[0].name" and the message.
func splitFailure(f string) (path, msg string) {
	loc, msg, ok := strings.Cut(f, ": ")
	if !ok {
		return "", f
	}

	if loc == "(root)" {
		return "", msg
	}

	var b strings.Builder

	for i, seg := range strings.Split(loc, ".") {
		if _, err := strconv.Atoi(seg); err == nil {
			b.WriteString("[" + seg + "]")
			continue
		}

		if i > 0 {
			b.WriteString(".")
		}

		b.WriteString(seg)
	}

	return b.String(), msg
}

var (
	additionalRE = regexp.MustCompile(`additional propert(?:y|ies) (.*) not allowed`)
	quotedRE     = regexp.MustCompile(`'([^']*)'`)
)

// unknownFields reads the field names out of a "not a key" failure.
func unknownFields(msg string) (keys []string, ok bool) {
	if msg == "not a key this service reads" {
		return []string{""}, true
	}

	m := additionalRE.FindStringSubmatch(msg)
	if m == nil {
		return nil, false
	}

	for _, q := range quotedRE.FindAllStringSubmatch(m[1], -1) {
		keys = append(keys, q[1])
	}

	return keys, len(keys) > 0
}

func join(path, key string) string {
	switch {
	case key == "":
		return path
	case path == "":
		return key
	}

	return path + "." + key
}

// normalize makes every list non-nil, so that an omitted section and an
// empty one render alike.
func (b *Bundle) normalize() {
	if b.Projects == nil {
		b.Projects = []Project{}
	}

	for i := range b.Projects {
		if b.Projects[i].Endpoints == nil {
			b.Projects[i].Endpoints = []Endpoint{}
		}
	}

	if b.Clients == nil {
		b.Clients = []Client{}
	}

	if b.Access.Groups == nil {
		b.Access.Groups = []Group{}
	}

	for i := range b.Access.Groups {
		if b.Access.Groups[i].Members == nil {
			b.Access.Groups[i].Members = []string{}
		}
	}

	if b.Access.Grants == nil {
		b.Access.Grants = []Grant{}
	}

	if b.Alerts.Receivers == nil {
		b.Alerts.Receivers = []Receiver{}
	}

	if b.Alerts.Routes == nil {
		b.Alerts.Routes = []Route{}
	}

	if b.Status == nil {
		b.Status = []StatusEntity{}
	}

	if b.CI == nil {
		b.CI = []CIRequest{}
	}

	if b.Registry == nil {
		b.Registry = []RegistryRow{}
	}
}
