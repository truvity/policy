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
		PasswordSecret string `json:"passwordSecret"`
		MaxConnections int    `json:"maxConnections"`
	} `json:"database"`
	Secrets config.SecretsSource `json:"secrets"`
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
	if cfg.Database.PasswordSecret != "db/password" || cfg.Secrets.Source != "file" {
		t.Errorf("database.passwordSecret = %q, secrets.source = %q", cfg.Database.PasswordSecret, cfg.Secrets.Source)
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

// The database URL is the other place a password hides: in the user
// information, or as a query parameter the driver reads like any other. The
// fragment refuses both, and the error names the key and never the value.
func TestAPasswordInTheDatabaseURLIsRefusedAndNotEchoed(t *testing.T) {
	for _, fixture := range []string{"password-in-url-query.yaml", "password-in-url-userinfo.yaml"} {
		t.Run(fixture, func(t *testing.T) {
			var cfg shortener
			err := config.Load("testdata/"+fixture, schema(t), &cfg)
			if err == nil {
				t.Fatal("a database URL carrying a password was accepted")
			}
			if strings.Contains(err.Error(), "hunter2") {
				t.Fatalf("the error quoted the value it refused:\n%v", err)
			}
			if !strings.Contains(err.Error(), "database.url") {
				t.Errorf("the error does not name the offending key:\n%v", err)
			}
		})
	}
}

// passfile names a file, which is how the contract says a secret may arrive.
// Refusing it with the password would push a service back to the
// environment for no reason.
func TestAPassfileInTheDatabaseURLIsAccepted(t *testing.T) {
	var cfg shortener
	if err := config.Load("testdata/passfile-in-url.yaml", schema(t), &cfg); err != nil {
		t.Fatalf("a database URL naming a password file was refused: %v", err)
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
	// The same fixture the TypeScript loader is tested against: two loaders
	// that claim to implement one contract must refuse the same documents.
	var cfg shortener
	if err := config.Load("testdata/empty.yaml", schema(t), &cfg); err == nil {
		t.Fatal("an empty configuration file was accepted, which would start the service on defaults nobody chose")
	}
}

// `...Env` is retired as a spelling (config.md rule 5, decision 0012): a
// document that still carries one is refused, and the error names the key.
func TestARetiredEnvSpellingIsRefused(t *testing.T) {
	var cfg shortener
	err := config.Load("testdata/env-spelling.yaml", schema(t), &cfg)
	if err == nil {
		t.Fatal("a database.passwordEnv was accepted")
	}
	if !strings.Contains(err.Error(), "passwordEnv") {
		t.Errorf("the error does not name the retired key:\n%v", err)
	}
}

func TestASecretNameThatClimbsIsRefused(t *testing.T) {
	var cfg shortener
	err := config.Load("testdata/secret-name-climbs.yaml", schema(t), &cfg)
	if err == nil {
		t.Fatal("a secret name that climbs out of its root was accepted")
	}
	if strings.Contains(err.Error(), "../") {
		t.Errorf("the error quotes the name it refused:\n%v", err)
	}
}
