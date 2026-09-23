// Package config loads a service's configuration: read one file, validate it
// against a schema, decode it into a type, and stop.
//
// That is the whole of it, deliberately. There is no lifecycle here, no
// dependency wiring, no HTTP, no reflection over the environment — those are
// the things a configuration package grows into when nobody says it must not,
// and a service that depends on them cannot be understood without them.
//
// The contract this implements is docs/contracts/config.md.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	yaml "go.yaml.in/yaml/v3"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/truvity/policy"
)

// Load reads the configuration file at path, validates it against schema,
// and decodes it into v, which must be a non-nil pointer.
//
// The file is YAML, which means JSON is accepted too. Validation happens
// BEFORE decoding, so a service never sees a value the schema would have
// rejected, and an error names the path that failed rather than the file.
//
// schema is the service's own schema. Any `$ref` to a shape this repository
// publishes resolves from the embedded copies; nothing is fetched.
func Load(filePath string, schema []byte, v any) error {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return &Error{File: filePath, Err: err}
	}

	var doc any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return &Error{File: filePath, Err: fmt.Errorf("not valid YAML: %w", err)}
	}
	if doc == nil {
		return &Error{File: filePath, Err: errors.New("file is empty")}
	}

	// Round-trip through JSON so that validation and decoding see exactly the
	// same document: the schema is a JSON Schema, and the struct tags a
	// service writes are JSON tags.
	asJSON, err := json.Marshal(doc)
	if err != nil {
		return &Error{File: filePath, Err: fmt.Errorf("cannot be represented as JSON: %w", err)}
	}
	var normalised any
	if err := json.Unmarshal(asJSON, &normalised); err != nil {
		return &Error{File: filePath, Err: err}
	}

	if err := Validate(normalised, schema); err != nil {
		var ve *Error
		if errors.As(err, &ve) {
			ve.File = filePath
		}
		return err
	}

	if err := json.Unmarshal(asJSON, v); err != nil {
		return &Error{File: filePath, Err: fmt.Errorf("valid against the schema but does not fit %T: %w", v, err)}
	}
	return nil
}

// Validate checks an already-decoded document against schema. Load calls it;
// it is exported because a chart's tests validate what they render with the
// same call, which is what stops the two drifting.
func Validate(doc any, schema []byte) error {
	compiled, err := compile(schema)
	if err != nil {
		return &Error{Err: err}
	}
	if err := compiled.Validate(doc); err != nil {
		var ve *jsonschema.ValidationError
		if errors.As(err, &ve) {
			return &Error{Failures: flatten(ve)}
		}
		return &Error{Err: err}
	}
	return nil
}

func compile(schema []byte) (*jsonschema.Schema, error) {
	doc, err := jsonschema.UnmarshalJSON(strings.NewReader(string(schema)))
	if err != nil {
		return nil, fmt.Errorf("the schema itself is not valid JSON: %w", err)
	}

	c := jsonschema.NewCompiler()

	// The shapes this repository publishes, by the `$id` they carry. Loaded
	// eagerly: there are a handful, they are small, and a lazy loader would
	// be a network fetch waiting to be added by someone in a hurry.
	if err := fs.WalkDir(policy.Schemas, "schemas", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || path.Ext(p) != ".json" {
			return err
		}
		b, err := policy.Schemas.ReadFile(p)
		if err != nil {
			return err
		}
		shared, err := jsonschema.UnmarshalJSON(strings.NewReader(string(b)))
		if err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		id := policy.SchemaBase + strings.TrimPrefix(p, "schemas/")
		return c.AddResource(id, shared)
	}); err != nil {
		return nil, err
	}

	const self = "service://config"
	if err := c.AddResource(self, doc); err != nil {
		return nil, err
	}
	return c.Compile(self)
}

// flatten turns the validation error tree into one line per failure, each
// naming the path in the document.
//
// It walks to the LEAVES rather than taking the library's own flat output.
// That output collapses a failure behind a `$ref` into "validation failed"
// against the referencing key, which is exactly the case a strict schema
// built from shared fragments produces: the useful message — which key is
// not allowed — is the cause it drops. A service refusing to start must say
// which key was wrong, because the person reading is looking at a file.
func flatten(ve *jsonschema.ValidationError) []string {
	printer := message.NewPrinter(language.English)

	var out []string
	var walk func(*jsonschema.ValidationError)
	walk = func(e *jsonschema.ValidationError) {
		if len(e.Causes) > 0 {
			for _, c := range e.Causes {
				walk(c)
			}
			return
		}
		loc := strings.Join(e.InstanceLocation, ".")
		if loc == "" {
			loc = "(root)"
		}
		msg := e.ErrorKind.LocalizedString(printer)
		// What a strict schema says when a key is not in it. Accurate,
		// and meaningless to anyone who has not read the specification:
		// this is the most common configuration mistake there is, so its
		// message is the one worth translating.
		if msg == "false schema" {
			msg = "not a key this service reads"
		}
		out = append(out, loc+": "+msg)
	}
	walk(ve)

	sort.Strings(out)
	return dedupe(out)
}

func dedupe(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := in[:0]
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// Secret reads the environment variable named by the configuration.
//
// A configuration file carries the NAME of the variable, never the value:
// files are rendered into config maps, printed when somebody debugs a
// deployment, and committed as test fixtures, and a secret has to survive all
// three being true.
//
// An unset or empty variable is an error, and the error names the variable
// rather than quoting anything.
func Secret(name string) (string, error) {
	if name == "" {
		return "", errors.New("no environment variable was named for this secret")
	}
	v, ok := os.LookupEnv(name)
	if !ok {
		return "", fmt.Errorf("environment variable %s is not set", name)
	}
	if v == "" {
		return "", fmt.Errorf("environment variable %s is empty", name)
	}
	return v, nil
}

// Error is what Load and Validate return. It names the file and every failing
// path; it never contains a value from the file, because an error is logged
// and a configuration file may sit next to a secret's name.
type Error struct {
	File     string
	Failures []string
	Err      error
}

func (e *Error) Error() string {
	var b strings.Builder
	b.WriteString("configuration")
	if e.File != "" {
		b.WriteString(" " + e.File)
	}
	switch {
	case len(e.Failures) > 0:
		b.WriteString(" is not valid:")
		for _, f := range e.Failures {
			b.WriteString("\n  " + f)
		}
	case e.Err != nil:
		b.WriteString(": " + e.Err.Error())
	}
	return b.String()
}

func (e *Error) Unwrap() error { return e.Err }
