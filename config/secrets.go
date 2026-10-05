package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
)

// The sources a `secrets` block names (docs/contracts/config.md rule 5).
const (
	SourceEnv     = "env"
	SourceFile    = "file"
	SourceSSM     = "ssm"
	SourceOpenBao = "openbao"
)

// lambdaEnv is the variable the Lambda runtime sets in every function. A
// function's environment is not a place for a secret (it is shown by the
// console and the API and kept in every version), so a process that finds it
// set refuses to read a secret from the environment, whatever the file says.
const lambdaEnv = "AWS_LAMBDA_FUNCTION_NAME"

// SecretsSource is the decoded `secrets` block: the ONE source a service reads
// every `...Secret` field through, and where that source looks. Embed or
// reference it from the service's own type, beside its other configuration;
// its schema is schemas/fragments/secrets.json.
type SecretsSource struct {
	Source string `json:"source"`
	Root   string `json:"root,omitempty"`
}

// rootSegment is one segment of a root: no empty segment and no `.` or `..`,
// so path cleaning cannot move a name out from under it.
var rootSegment = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]*$`)

// secretName is a name under a root: relative, made of segments that start with
// a letter, a digit or an underscore, so that it cannot be `..` and cannot
// start at `/`. It is the pattern of schemas/fragments/secrets.json `$defs/name`.
var secretName = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]*(/[A-Za-z0-9_][A-Za-z0-9_.-]*)*$`)

var envName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// cleanRoot says whether root, after an optional leading slash, is made of
// rootSegments.
func cleanRoot(root string) bool {
	if root == "" || root == "/" {
		return false
	}
	for _, seg := range strings.Split(strings.TrimPrefix(root, "/"), "/") {
		if !rootSegment.MatchString(seg) {
			return false
		}
	}
	return true
}

// Check holds the block to what the schema's shape cannot say, and defaults an
// empty Source to env (an absent `secrets` block is the environment). It does
// not look at the platform: the refusal of env on a serverless function is
// made by [Secrets.Get], where the process is.
func (s *SecretsSource) Check() error {
	if s.Source == "" {
		s.Source = SourceEnv
	}
	switch s.Source {
	case SourceEnv:
		if s.Root != "" {
			return errors.New("secrets.root is for the sources file, ssm and openbao: an environment variable has no root")
		}
		if os.Getenv(lambdaEnv) != "" {
			return errLambdaEnv
		}
	case SourceFile:
		if !strings.HasPrefix(s.Root, "/") || !filepath.IsAbs(s.Root) || !cleanRoot(s.Root) {
			return errors.New("secrets.root must be an absolute directory with source file, with no empty, . or .. segment")
		}
	case SourceSSM:
		if !strings.HasPrefix(s.Root, "/") || !cleanRoot(s.Root) {
			return errors.New("secrets.root must be an absolute parameter path with source ssm, such as /audit/main/private/config, with no empty, . or .. segment and no trailing slash")
		}
	case SourceOpenBao:
		if strings.HasPrefix(s.Root, "/") || !cleanRoot(s.Root) {
			return errors.New("secrets.root must be the mount path with source openbao, with no leading slash and no empty, . or .. segment " +
				"(for a KV v2 mount, include the data segment: secret/data/<app>)")
		}
	default:
		return fmt.Errorf("secrets.source is %q and must be env, file, ssm or openbao", s.Source)
	}
	return nil
}

// Store reads one secret from a remote store by its full path. The ssm and
// openbao sources are one: this package carries neither client (a service that
// uses neither should not link them), so the service supplies the store with
// [WithStore], backed by its own SSM or OpenBao client, with the process's own
// identity. A store reads the credential live, so a rotated value is seen by the
// next call; the configuration that names it is not re-read.
//
// A Store must not put the path or a value in the errors it returns: Get does
// not print them, but a log of the wrapped cause would. It may return
// [ErrNotFound], or wrap it, for a secret that does not exist.
type Store interface {
	Get(ctx context.Context, path string) (string, error)
}

// ErrNotFound is what a [Store] returns, or wraps, when the secret does not
// exist. [errors.Is] finds it through the error [Secrets.Get] returns.
var ErrNotFound = errors.New("secret not found")

// storeError is a failed read from a Store. It prints the field's source and
// root and never the cause, which a store's client may fill with the path it
// asked for; the cause stays reachable with errors.Is and errors.As.
type storeError struct {
	source, root string
	cause        error
}

func (e *storeError) Error() string {
	what := "could not be read"
	if errors.Is(e.cause, ErrNotFound) {
		what = "does not exist"
	}
	return fmt.Sprintf("the secret it names under secrets.root %s %s (secrets.source is %s)", e.root, what, e.source)
}

func (e *storeError) Unwrap() error { return e.cause }

// Option configures [NewSecrets].
type Option func(*Secrets)

// WithStore supplies the store for a remote source (SourceSSM or SourceOpenBao).
//
// A nil store, including a nil pointer in a Store, is not registered, so
// [NewSecrets] refuses the source instead of failing on first use.
func WithStore(source string, s Store) Option {
	return func(r *Secrets) {
		if v := reflect.ValueOf(s); s == nil || (v.Kind() == reflect.Pointer || v.Kind() == reflect.Map || v.Kind() == reflect.Func || v.Kind() == reflect.Interface || v.Kind() == reflect.Slice) && v.IsNil() {
			return
		}
		if r.stores == nil {
			r.stores = map[string]Store{}
		}
		r.stores[source] = s
	}
}

// Secrets resolves the name a field holds to the secret it stands for, through
// the one source the file declares. Safe for concurrent use.
type Secrets struct {
	src    SecretsSource
	stores map[string]Store
}

// NewSecrets is the resolver for a `secrets` block. It checks the block and
// refuses a remote source with no store, at start-up, before the first Get.
func NewSecrets(src SecretsSource, opts ...Option) (*Secrets, error) {
	if err := src.Check(); err != nil {
		return nil, &Error{Err: err}
	}
	s := &Secrets{src: src}
	for _, o := range opts {
		o(s)
	}
	if (src.Source == SourceSSM || src.Source == SourceOpenBao) && s.stores[src.Source] == nil {
		return nil, &Error{Err: fmt.Errorf("secrets.source is %s and this binary was given no store for it", src.Source)}
	}
	return s, nil
}

// Source is the declared source.
func (s *Secrets) Source() string { return s.src.Source }

// Get reads the secret `name` that `field` holds. field is the key as the file
// spells it (`database.passwordSecret`). An error names the field and the
// source and root, and never the name or a value: an error is logged and
// rendered, and a refusal that quoted what it was given would be one more place
// a secret pasted in place of its name reaches a log.
//
// On a serverless function (AWS_LAMBDA_FUNCTION_NAME set) the env source is
// refused whatever the file says: the function's environment is capped and
// shown in plaintext in the console.
func (s *Secrets) Get(ctx context.Context, field, name string) (string, error) {
	v, err := s.get(ctx, name)
	if err != nil {
		return "", fmt.Errorf("%s: %w", field, err)
	}
	return v, nil
}

func (s *Secrets) get(ctx context.Context, name string) (string, error) {
	switch s.src.Source {
	case SourceEnv:
		if os.Getenv(lambdaEnv) != "" {
			return "", errLambdaEnv
		}
		if !envName.MatchString(name) {
			return "", errors.New("the name is not that of an environment variable (secrets.source is env)")
		}
		v, ok := os.LookupEnv(name)
		switch {
		case !ok:
			return "", errors.New("the environment variable it names is not set (secrets.source is env)")
		case v == "":
			return "", errors.New("the environment variable it names is empty (secrets.source is env)")
		}
		return v, nil
	case SourceFile:
		if !secretName.MatchString(name) {
			return "", errNotAName
		}
		b, err := os.ReadFile(filepath.Join(s.src.Root, filepath.FromSlash(name)))
		if err != nil {
			return "", fmt.Errorf("the secret it names is not readable under secrets.root %s: %w", s.src.Root, pathError(err))
		}
		v := strings.TrimSuffix(string(b), "\n")
		if v == "" {
			return "", fmt.Errorf("the secret it names is empty (a file under secrets.root %s)", s.src.Root)
		}
		return v, nil
	case SourceSSM, SourceOpenBao:
		if !secretName.MatchString(name) {
			return "", errNotAName
		}
		st := s.stores[s.src.Source]
		if st == nil {
			return "", fmt.Errorf("secrets.source is %s and no store was supplied", s.src.Source)
		}
		v, err := st.Get(ctx, path.Join(s.src.Root, name))
		if err != nil {
			return "", &storeError{source: s.src.Source, root: s.src.Root, cause: err}
		}
		if v == "" {
			return "", fmt.Errorf("the secret it names under secrets.root %s is empty (secrets.source is %s)", s.src.Root, s.src.Source)
		}
		return v, nil
	}
	return "", fmt.Errorf("secrets.source %q is not env, file, ssm or openbao", s.src.Source)
}

var errLambdaEnv = errors.New("secrets.source is env on AWS Lambda, where a function's environment is not a place for a secret: use secrets.source ssm or file")

var errNotAName = errors.New("the name is not a secret name: a relative path of letters, digits, dots, underscores and dashes, which does not climb")

// pathError drops the path an *os.PathError repeats.
func pathError(err error) error {
	var pe *os.PathError
	if errors.As(err, &pe) {
		return pe.Err
	}
	return err
}
