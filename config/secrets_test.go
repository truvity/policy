package config_test

import (
	"context"
	"errors"
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

func TestSecretsEnvIsRefusedOnLambda(t *testing.T) {
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "f")
	t.Setenv("EXAMPLE_PASSWORD", "value")
	_, err := resolver(t, config.SecretsSource{Source: "env"}).Get(context.Background(), "k", "EXAMPLE_PASSWORD")
	if err == nil {
		t.Fatal("the env source was read on a function")
	}
	if strings.Contains(err.Error(), "value") {
		t.Errorf("the error carries the value: %v", err)
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
