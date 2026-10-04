package config

import (
	"fmt"
	"regexp"
	"strconv"
)

// APIVersionKey is the key a document names its version under, at its root:
// `apiVersion: <group>/<kind>/v<N>`. Absent means v1.
const APIVersionKey = "apiVersion"

// The form of an apiVersion: a group (a DNS-like name), a kind, and a
// version with no leading zero. The envelope schema (schemas/service.json)
// holds the same pattern, so a chart's tests refuse what the loader refuses.
var apiVersionForm = regexp.MustCompile(`^([a-z0-9]([a-z0-9.-]*[a-z0-9])?/[a-z0-9]([a-z0-9-]*[a-z0-9])?)/v([1-9][0-9]*)$`)

// The form of a kind's name: an apiVersion without its version.
var kindForm = regexp.MustCompile(`^[a-z0-9]([a-z0-9.-]*[a-z0-9])?/[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// Kind is one kind of configuration document, and the versions of it a
// binary reads: the current one, N, and optionally the one before, N-1.
//
// Two and no more, deliberately (docs/contracts/config.md, rule 7). A binary
// that reads N-1 can be rolled out before its configuration moves to N, which
// is what makes a version change two ordinary deployments instead of one
// synchronised one. A binary that read every version would carry every
// conversion forever, and nobody would ever be able to say which documents
// are still in use.
//
// Each version has its own authored schema; nothing here generates one.
type Kind struct {
	// Name is "<group>/<kind>": the apiVersion without its version, for
	// example "example.com/notifier".
	Name string

	// Version is N, the version this binary is written against. Zero means 1.
	Version int

	// Schema is version N's schema, and the type v in LoadKind is version N's
	// type.
	Schema []byte

	// Previous is version N-1's schema, or nil when this binary reads N alone.
	Previous []byte

	// Upgrade turns a document that is valid against Previous into one of
	// version N. It is given the whole document, apiVersion included, and
	// returns the converted one; the loader then sets apiVersion to N and
	// validates the result against Schema, so a conversion that produces
	// something N does not accept is refused rather than decoded. Required
	// when Previous is set.
	//
	// An error it returns is reported with the file's name, and is logged:
	// it must name keys, never quote values.
	Upgrade func(doc map[string]any) (map[string]any, error)
}

// LoadKind reads the configuration file at filePath as a document of kind,
// and decodes it into v, which must be a non-nil pointer to version N's type.
//
// It reads the document's apiVersion before anything else and chooses the
// schema by it:
//
//   - version N is validated against Schema and decoded;
//   - version N-1, when the binary reads it, is validated against Previous,
//     converted by Upgrade, validated against Schema, and decoded;
//   - anything else is refused, naming the key: a version newer than N (a
//     binary older than its configuration), older than the oldest this binary
//     reads, another kind of document, or a value not of the form at all.
//
// A document with no apiVersion is version 1.
func LoadKind(filePath string, kind Kind, v any) error {
	if err := kind.check(); err != nil {
		return &Error{File: filePath, Err: err}
	}

	doc, err := read(filePath)
	if err != nil {
		return err
	}
	if failure := versionFailure(doc, kind); failure != "" {
		return &Error{File: filePath, Failures: []string{failure}}
	}

	n := kind.version()
	current := fmt.Sprintf("%s/v%d", kind.Name, n)
	// A root that is not a mapping is checked against N's schema, which
	// refuses it naming the root: there is no version to choose by.
	if _, isMap := doc.(map[string]any); !isMap || documentVersion(doc) == n {
		if err := Validate(doc, kind.Schema); err != nil {
			return inFile(err, filePath, "as "+current)
		}
		return decode(filePath, doc, v)
	}

	// N-1: versionFailure has already refused everything else.
	previous := fmt.Sprintf("%s/v%d", kind.Name, n-1)
	if err := Validate(doc, kind.Previous); err != nil {
		return inFile(err, filePath, "as "+previous)
	}
	// Validation has just proved the root is an object.
	upgraded, err := kind.Upgrade(doc.(map[string]any))
	if err != nil {
		return &Error{File: filePath, Err: fmt.Errorf("upgrading v%d to v%d: %w", n-1, n, err)}
	}
	if upgraded == nil {
		return &Error{File: filePath, Err: fmt.Errorf("upgrading v%d to v%d returned no document", n-1, n)}
	}
	upgraded[APIVersionKey] = current
	if err := Validate(any(upgraded), kind.Schema); err != nil {
		return inFile(err, filePath, fmt.Sprintf("upgraded from v%d to %s", n-1, current))
	}
	return decode(filePath, upgraded, v)
}

func (k Kind) version() int {
	if k.Version == 0 {
		return 1
	}
	return k.Version
}

// check refuses a declaration that cannot work. It is the binary's mistake,
// not the file's, so it is reported before the file is read.
func (k Kind) check() error {
	switch {
	case !kindForm.MatchString(k.Name):
		return fmt.Errorf("the binary declares kind %q, which is not of the form <group>/<kind>", k.Name)
	case k.Version < 0:
		return fmt.Errorf("the binary declares %s at version %d", k.Name, k.Version)
	case len(k.Schema) == 0:
		return fmt.Errorf("the binary declares %s with no schema", k.Name)
	case k.Previous != nil && k.version() < 2:
		return fmt.Errorf("the binary declares %s v1 with a previous version", k.Name)
	case k.Previous != nil && k.Upgrade == nil:
		return fmt.Errorf("the binary declares %s v%d with a previous version and no upgrade", k.Name, k.version())
	case k.Previous == nil && k.Upgrade != nil:
		return fmt.Errorf("the binary declares %s v%d with an upgrade and no previous version", k.Name, k.version())
	}
	return nil
}

// documentVersion is the version a document says it is: 1 when it says
// nothing, 0 when what it says is not of the form.
func documentVersion(doc any) int {
	m, ok := doc.(map[string]any)
	if !ok {
		return 1
	}
	raw, present := m[APIVersionKey]
	if !present {
		return 1
	}
	s, ok := raw.(string)
	if !ok {
		return 0
	}
	match := apiVersionForm.FindStringSubmatch(s)
	if match == nil {
		return 0
	}
	n, err := strconv.Atoi(match[4])
	if err != nil {
		return 0
	}
	return n
}

// versionFailure says why a document's apiVersion is not one kind reads, or
// returns "" when it is. A kind with no Name is the plain loader's: it reads
// v1 of whatever the document is.
//
// The message names the key and the version. It never quotes the value
// otherwise: a value that is not of the form could be anything.
func versionFailure(doc any, kind Kind) string {
	const key = APIVersionKey + ": "

	n := kind.version()
	reads := fmt.Sprintf("v%d", n)
	oldest := n
	if kind.Previous != nil {
		reads += fmt.Sprintf(", v%d", n-1)
		oldest = n - 1
	}

	m, ok := doc.(map[string]any)
	if !ok {
		// Not a mapping at all. The schema says so, naming the root.
		return ""
	}
	raw, present := m[APIVersionKey]
	if !present {
		if oldest > 1 {
			return key + "absent, which is v1, older than this binary reads (" + reads + ")"
		}
		return ""
	}

	s, ok := raw.(string)
	match := apiVersionForm.FindStringSubmatch(s)
	if !ok || match == nil {
		return key + "not of the form <group>/<kind>/v<N>"
	}
	if kind.Name != "" && match[1] != kind.Name {
		return key + "names another kind of document; this binary reads " + kind.Name
	}
	got := documentVersion(doc)
	switch {
	case got > n:
		return fmt.Sprintf("%sv%d is newer than this binary reads (%s)", key, got, reads)
	case got < oldest:
		return fmt.Sprintf("%sv%d is older than this binary reads (%s)", key, got, reads)
	}
	return ""
}
