package config_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/truvity/policy/config"
)

// The table every loader that resolves the path runs, from the shared
// fixtures (config/testdata/path-from.json).
func TestTheSharedPathCases(t *testing.T) {
	raw, err := os.ReadFile("testdata/path-from.json")
	if err != nil {
		t.Fatal(err)
	}
	var table struct {
		Env   string `json:"env"`
		Cases []struct {
			Name    string            `json:"name"`
			Args    []string          `json:"args"`
			Env     map[string]string `json:"env"`
			Want    string            `json:"want"`
			Refused []string          `json:"refused"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &table); err != nil {
		t.Fatal(err)
	}
	if len(table.Cases) == 0 || table.Env == "" {
		t.Fatal("the shared path cases decoded to nothing")
	}

	for _, c := range table.Cases {
		t.Run(c.Name, func(t *testing.T) {
			// t.Setenv then Unsetenv: the variable is restored after the
			// test whichever state it started in, and is absent unless the
			// case sets it.
			t.Setenv(table.Env, "")
			if err := os.Unsetenv(table.Env); err != nil {
				t.Fatal(err)
			}
			for k, v := range c.Env {
				t.Setenv(k, v)
			}

			got, err := config.PathFrom(c.Args, table.Env)
			if len(c.Refused) > 0 {
				if err == nil {
					t.Fatalf("accepted %q; want refused with %q", got, c.Refused)
				}
				for _, want := range c.Refused {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("the refusal does not say %q:\n%v", want, err)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("refused: %v", err)
			}
			if got != c.Want {
				t.Errorf("got %q, want %q", got, c.Want)
			}
		})
	}
}
