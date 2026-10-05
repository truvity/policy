package config_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/truvity/policy/config"
)

type fakeStore map[string]string

func (f fakeStore) Get(_ context.Context, p string) (string, error) {
	v, ok := f[p]
	if !ok {
		return "", errors.New("not found")
	}
	return v, nil
}

func resolver(t *testing.T, src config.SecretsSource, opts ...config.Option) *config.Secrets {
	t.Helper()
	s, err := config.NewSecrets(src, opts...)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestSecretsEnvSource(t *testing.T) {
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "")
	t.Setenv("EXAMPLE_PASSWORD", "value")
	t.Setenv("EXAMPLE_EMPTY", "")
	s := resolver(t, config.SecretsSource{})
	if s.Source() != config.SourceEnv {
		t.Errorf("an absent source is %q, want env", s.Source())
	}
	got, err := s.Get(context.Background(), "database.passwordSecret", "EXAMPLE_PASSWORD")
	if err != nil || got != "value" {
		t.Fatalf("got %q, %v", got, err)
	}
	for name, n := range map[string]string{"unset": "EXAMPLE_NOT_SET", "empty": "EXAMPLE_EMPTY", "not a variable name": "a/b", "no name": ""} {
		_, err := s.Get(context.Background(), "database.passwordSecret", n)
		if err == nil {
			t.Fatalf("%s: accepted", name)
		}
		if !strings.Contains(err.Error(), "database.passwordSecret") {
			t.Errorf("%s: the error does not name the field: %v", name, err)
		}
	}
}

func TestSecretsEnvIsRefusedOnLambdaAtConstruction(t *testing.T) {
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "f")
	t.Setenv("EXAMPLE_PASSWORD", "value")
	for _, src := range []config.SecretsSource{{}, {Source: "env"}} {
		if _, err := config.NewSecrets(src); err == nil {
			t.Fatalf("the env source %+v was accepted on a function", src)
		}
	}
	src := config.SecretsSource{Source: "env"}
	if err := src.Check(); err == nil || strings.Contains(err.Error(), "value") {
		t.Errorf("Check on a function: %v", err)
	}
}

func TestSecretsFileSource(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "db"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "db", "password"), []byte("hunter2\n"), 0o440); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "empty"), nil, 0o440); err != nil {
		t.Fatal(err)
	}
	s := resolver(t, config.SecretsSource{Source: "file", Root: dir})
	got, err := s.Get(context.Background(), "k", "db/password")
	if err != nil || got != "hunter2" {
		t.Fatalf("got %q, %v", got, err)
	}
	for name, n := range map[string]string{"climbs": "../x", "absolute": "/etc/passwd", "empty segment": "db//password", "missing": "nope", "empty file": "empty", "dot": "."} {
		_, err := s.Get(context.Background(), "k", n)
		if err == nil {
			t.Errorf("%s: accepted", name)
			continue
		}
		if n != "" && strings.Contains(err.Error(), n) && n != "nope" && n != "empty" {
			t.Errorf("%s: the error quotes the name: %v", name, err)
		}
	}
}

func TestSecretsRemoteSources(t *testing.T) {
	st := fakeStore{"/audit/main/db/password": "v", "secret/audit/token": "t"}
	ssm := resolver(t, config.SecretsSource{Source: "ssm", Root: "/audit/main"}, config.WithStore("ssm", st))
	if got, err := ssm.Get(context.Background(), "k", "db/password"); err != nil || got != "v" {
		t.Fatalf("ssm: %q, %v", got, err)
	}
	if _, err := ssm.Get(context.Background(), "k", "../x"); err == nil {
		t.Error("ssm: a climbing name was accepted")
	}
	bao := resolver(t, config.SecretsSource{Source: "openbao", Root: "secret"}, config.WithStore("openbao", st))
	if got, err := bao.Get(context.Background(), "k", "audit/token"); err != nil || got != "t" {
		t.Fatalf("openbao: %q, %v", got, err)
	}
	if _, err := config.NewSecrets(config.SecretsSource{Source: "ssm", Root: "/audit"}); err == nil {
		t.Error("a remote source with no store was accepted")
	}
}

func TestSecretsSourceCheck(t *testing.T) {
	bad := map[string]config.SecretsSource{
		"env with a root":     {Source: "env", Root: "/x"},
		"file, relative root": {Source: "file", Root: "secrets"},
		"file, no root":       {Source: "file"},
		"file, dot-dot":       {Source: "file", Root: "/run/../etc"},
		"file, empty segment": {Source: "file", Root: "/run//s"},
		"file, dot segment":   {Source: "file", Root: "/run/./s"},
		"file, root is /":     {Source: "file", Root: "/"},
		"ssm, relative":       {Source: "ssm", Root: "audit"},
		"ssm, trailing slash": {Source: "ssm", Root: "/audit/"},
		"openbao, no mount":   {Source: "openbao"},
		"openbao, climbing":   {Source: "openbao", Root: "secret/.."},
		"an unknown source":   {Source: "vault", Root: "/x"},
	}
	for name, src := range bad {
		if err := src.Check(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	good := []config.SecretsSource{{}, {Source: "env"}, {Source: "file", Root: "/var/run/secrets/x"}, {Source: "ssm", Root: "/a/b"}, {Source: "openbao", Root: "secret/x"}}
	for _, src := range good {
		if err := src.Check(); err != nil {
			t.Errorf("%+v: %v", src, err)
		}
	}
}

type leakyStore struct{ err error }

func (l leakyStore) Get(context.Context, string) (string, error) { return "", l.err }

func TestAStoreErrorIsNotPrintedButIsReachable(t *testing.T) {
	leak := errors.New("GetParameter /audit/main/db/password: AccessDenied")
	s := resolver(t, config.SecretsSource{Source: "ssm", Root: "/audit/main"}, config.WithStore("ssm", leakyStore{leak}))
	_, err := s.Get(context.Background(), "database.passwordSecret", "db/password")
	if err == nil {
		t.Fatal("a failing store was accepted")
	}
	for _, bad := range []string{"db/password", "AccessDenied", "GetParameter"} {
		if strings.Contains(err.Error(), bad) {
			t.Errorf("the error carries %q: %v", bad, err)
		}
	}
	if !strings.Contains(err.Error(), "database.passwordSecret") || !errors.Is(err, leak) {
		t.Errorf("the error must name the field and keep the cause for errors.Is: %v", err)
	}

	nf := resolver(t, config.SecretsSource{Source: "ssm", Root: "/audit"}, config.WithStore("ssm", leakyStore{fmt.Errorf("ParameterNotFound /audit/x: %w", config.ErrNotFound)}))
	_, err = nf.Get(context.Background(), "k", "x")
	if !errors.Is(err, config.ErrNotFound) || strings.Contains(err.Error(), "ParameterNotFound") {
		t.Errorf("not found: %v", err)
	}
}

func TestATypedNilStoreIsNotRegistered(t *testing.T) {
	var nilStore *nilPtrStore
	if _, err := config.NewSecrets(config.SecretsSource{Source: "ssm", Root: "/a"}, config.WithStore("ssm", nilStore)); err == nil {
		t.Error("a typed nil store was accepted")
	}
}

type nilPtrStore struct{}

func (*nilPtrStore) Get(context.Context, string) (string, error) { return "", nil }

func TestAnOpenBaoRootHasNoLeadingSlash(t *testing.T) {
	src := config.SecretsSource{Source: "openbao", Root: "/secret/data/audit"}
	if err := src.Check(); err == nil {
		t.Error("a leading slash was accepted")
	}
}
