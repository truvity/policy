package config_test

import (
	"os"
	"strings"
	"testing"

	"github.com/truvity/policy/config"
)

// The configuration of the worked example, in miniature: the envelope the
// contract defines plus two fields of its own.
type shortener struct {
	Listen struct {
		Address string `json:"address"`
	} `json:"listen"`
	Probes struct {
		Address string `json:"address"`
	} `json:"probes"`
	Log struct {
		Level string `json:"level"`
	} `json:"log"`
	BaseURL  string `json:"baseURL"`
	Database struct {
		URL            string `json:"url"`
		PasswordEnv    string `json:"passwordEnv"`
		MaxConnections int    `json:"maxConnections"`
	} `json:"database"`
}

func schema(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/shortener.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestAValidFileLoadsAndDecodes(t *testing.T) {
	var cfg shortener
	if err := config.Load("testdata/valid.yaml", schema(t), &cfg); err != nil {
		t.Fatalf("a valid configuration was refused: %v", err)
	}

	// The fragments are resolved from the embedded copies, so a `$ref` to a
	// shape this repository publishes works with no network at all. If that
	// stopped being true, this would fail rather than quietly validating
	// nothing.
	if cfg.Probes.Address != ":7070" {
		t.Errorf("probes.address = %q, want \":7070\"", cfg.Probes.Address)
	}
	if cfg.Database.MaxConnections != 20 {
		t.Errorf("database.maxConnections = %d, want 20", cfg.Database.MaxConnections)
	}
	if cfg.BaseURL != "https://example.com" {
		t.Errorf("baseURL = %q", cfg.BaseURL)
	}
}

// The failure the whole contract exists to prevent: a key that means nothing,
// set by a deployment, doing nothing, with no signal but behaviour.
func TestAnUnknownKeyIsRefused(t *testing.T) {
	var cfg shortener
	err := config.Load("testdata/unknown-key.yaml", schema(t), &cfg)
	if err == nil {
		t.Fatal("a configuration with an unknown key was accepted")
	}
	if !strings.Contains(err.Error(), "databse") {
		t.Errorf("the error does not name the offending key, so nobody can fix it:\n%v", err)
	}
}

func TestAMissingRequiredKeyIsRefusedAndNamed(t *testing.T) {
	var cfg shortener
	err := config.Load("testdata/missing-required.yaml", schema(t), &cfg)
	if err == nil {
		t.Fatal("a configuration missing a required key was accepted")
	}
	if !strings.Contains(err.Error(), "database") {
		t.Errorf("the error does not name what is missing:\n%v", err)
	}
}

// "invalid config" is not an error message. The person reading it is looking
// at a file and needs the path.
func TestAWrongTypeNamesThePath(t *testing.T) {
	var cfg shortener
	err := config.Load("testdata/wrong-type.yaml", schema(t), &cfg)
	if err == nil {
		t.Fatal("a configuration with a string where a number belongs was accepted")
	}
	if !strings.Contains(err.Error(), "database.maxConnections") {
		t.Errorf("the error does not name the path that failed:\n%v", err)
	}
}

// A schema that is strict only at the top level accepts a secret smuggled
// into a nested object. This is the case the fragments' strictness exists
// for, and the error must not echo what it found.
func TestASecretInTheFileIsRefusedAndNotEchoed(t *testing.T) {
	var cfg shortener
	err := config.Load("testdata/secret-in-file.yaml", schema(t), &cfg)
	if err == nil {
		t.Fatal("a password written into the configuration file was accepted")
	}
	if strings.Contains(err.Error(), "hunter2") {
		t.Fatalf("the error quoted the value it refused, which puts it in every log that records the refusal:\n%v", err)
	}
	if !strings.Contains(err.Error(), "password") {
		t.Errorf("the error does not name the offending key:\n%v", err)
	}
}

func TestValidationHappensBeforeDecoding(t *testing.T) {
	// An invalid document must leave the caller's value untouched: a service
	// that half-decodes and then fails has a configuration nobody chose.
	cfg := shortener{BaseURL: "untouched"}
	_ = config.Load("testdata/wrong-type.yaml", schema(t), &cfg)
	if cfg.BaseURL != "untouched" {
		t.Errorf("the value was written to before validation failed: %q", cfg.BaseURL)
	}
}

func TestAMissingFileSaysWhichFile(t *testing.T) {
	var cfg shortener
	err := config.Load("testdata/there-is-no-such-file.yaml", schema(t), &cfg)
	if err == nil {
		t.Fatal("a missing configuration file was accepted")
	}
	if !strings.Contains(err.Error(), "there-is-no-such-file.yaml") {
		t.Errorf("the error does not name the file:\n%v", err)
	}
}

func TestAnEmptyFileIsRefused(t *testing.T) {
	empty := t.TempDir() + "/empty.yaml"
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	var cfg shortener
	if err := config.Load(empty, schema(t), &cfg); err == nil {
		t.Fatal("an empty configuration file was accepted, which would start the service on defaults nobody chose")
	}
}

func TestSecret(t *testing.T) {
	t.Run("set", func(t *testing.T) {
		t.Setenv("EXAMPLE_PASSWORD", "value")
		got, err := config.Secret("EXAMPLE_PASSWORD")
		if err != nil {
			t.Fatal(err)
		}
		if got != "value" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("unset names the variable", func(t *testing.T) {
		_, err := config.Secret("EXAMPLE_PASSWORD_THAT_IS_NOT_SET")
		if err == nil {
			t.Fatal("an unset secret was accepted, so the service would start without it")
		}
		if !strings.Contains(err.Error(), "EXAMPLE_PASSWORD_THAT_IS_NOT_SET") {
			t.Errorf("the error does not name the variable:\n%v", err)
		}
	})

	t.Run("empty is not set", func(t *testing.T) {
		t.Setenv("EXAMPLE_EMPTY", "")
		if _, err := config.Secret("EXAMPLE_EMPTY"); err == nil {
			t.Fatal("an empty secret was accepted; an empty password is a misconfiguration, not a password")
		}
	})

	t.Run("no name at all", func(t *testing.T) {
		if _, err := config.Secret(""); err == nil {
			t.Fatal("a secret with no variable named was accepted")
		}
	})
}
