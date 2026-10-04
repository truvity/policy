package config_test

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/truvity/policy/config"
)

// The table every loader that reads versions runs, from the shared fixtures
// (config/testdata/versions/cases.json). Its $comment says what each field
// means.
type versionCases struct {
	Kind  string `json:"kind"`
	Cases []struct {
		Name    string         `json:"name"`
		File    string         `json:"file"`
		Reads   string         `json:"reads"`
		Schema  string         `json:"schema"`
		Upgrade string         `json:"upgrade"`
		Want    map[string]any `json:"want"`
		Refused []string       `json:"refused"`
	} `json:"cases"`
}

const versions = "testdata/versions/"

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(versions + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// upgradeV1 is the notifier's own conversion, the one every loader's test
// writes in its own language: v1's `retries` becomes v2's `retry.attempts`,
// and a v1 document that said nothing gets v1's documented default of three.
func upgradeV1(doc map[string]any) (map[string]any, error) {
	attempts := any(float64(3))
	if r, ok := doc["retries"]; ok {
		attempts = r
	}
	delete(doc, "retries")
	doc["retry"] = map[string]any{"attempts": attempts}
	return doc, nil
}

func TestTheSharedVersionCases(t *testing.T) {
	var table versionCases
	if err := json.Unmarshal(readFixture(t, "cases.json"), &table); err != nil {
		t.Fatal(err)
	}
	if len(table.Cases) == 0 {
		// A table that failed to decode into anything would pass every
		// case it does not have.
		t.Fatal("the shared version cases decoded to nothing")
	}

	for _, c := range table.Cases {
		t.Run(c.Name, func(t *testing.T) {
			upgrade := upgradeV1
			switch c.Upgrade {
			case "":
			case "drops-retry":
				upgrade = func(doc map[string]any) (map[string]any, error) {
					delete(doc, "retries")
					return doc, nil
				}
			case "fails":
				upgrade = func(map[string]any) (map[string]any, error) {
					return nil, errors.New("retries cannot be converted")
				}
			default:
				t.Fatalf("unknown upgrade %q in the table", c.Upgrade)
			}

			var got map[string]any
			var err error
			switch c.Reads {
			case "both":
				err = config.LoadKind(versions+c.File, config.Kind{
					Name:     table.Kind,
					Version:  2,
					Schema:   readFixture(t, "notifier.v2.schema.json"),
					Previous: readFixture(t, "notifier.v1.schema.json"),
					Upgrade:  upgrade,
				}, &got)
			case "current":
				err = config.LoadKind(versions+c.File, config.Kind{
					Name:    table.Kind,
					Version: 2,
					Schema:  readFixture(t, "notifier.v2.schema.json"),
				}, &got)
			case "load":
				err = config.Load(versions+c.File, readFixture(t, c.Schema), &got)
			default:
				t.Fatalf("unknown reads %q in the table", c.Reads)
			}

			if len(c.Refused) > 0 {
				if err == nil {
					t.Fatalf("accepted; want refused with %q", c.Refused)
				}
				for _, want := range c.Refused {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("the refusal does not say %q:\n%v", want, err)
					}
				}
				// The fixtures carry nothing secret, but they do carry an
				// endpoint, and an error that quoted it would quote a
				// password the next time.
				if strings.Contains(err.Error(), "https://example.com/hook") {
					t.Errorf("the refusal quotes a value from the file:\n%v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("refused: %v", err)
			}
			for path, want := range c.Want {
				if v := at(got, path); !reflect.DeepEqual(v, want) {
					t.Errorf("%s = %#v, want %#v", path, v, want)
				}
			}
		})
	}
}

func at(doc map[string]any, path string) any {
	var cur any = doc
	for _, part := range strings.Split(path, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[part]
	}
	return cur
}

// The struct a binary at N decodes into is N's type, whichever version the
// file was: that is what lets the rest of the program know nothing about v1.
func TestAnUpgradedDocumentDecodesIntoTheCurrentType(t *testing.T) {
	type notifier struct {
		APIVersion string `json:"apiVersion"`
		Endpoint   string `json:"endpoint"`
		Retry      struct {
			Attempts int `json:"attempts"`
		} `json:"retry"`
	}
	var cfg notifier
	err := config.LoadKind(versions+"v1.yaml", config.Kind{
		Name:     "example.com/notifier",
		Version:  2,
		Schema:   readFixture(t, "notifier.v2.schema.json"),
		Previous: readFixture(t, "notifier.v1.schema.json"),
		Upgrade:  upgradeV1,
	}, &cfg)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Retry.Attempts != 4 || cfg.APIVersion != "example.com/notifier/v2" {
		t.Errorf("got %+v", cfg)
	}
}

// A declaration that cannot work is the binary's mistake, and it says so
// before the file is read: the file in these cases is fine.
func TestADeclarationThatCannotWorkIsRefused(t *testing.T) {
	v2 := readFixture(t, "notifier.v2.schema.json")
	v1 := readFixture(t, "notifier.v1.schema.json")
	for name, kind := range map[string]config.Kind{
		"no name":                  {Version: 2, Schema: v2},
		"a name with a version":    {Name: "example.com/notifier/v2", Version: 2, Schema: v2},
		"no schema":                {Name: "example.com/notifier", Version: 2},
		"a previous version of v1": {Name: "example.com/notifier", Version: 1, Schema: v1, Previous: v1, Upgrade: upgradeV1},
		"a previous and no upgrade": {
			Name: "example.com/notifier", Version: 2, Schema: v2, Previous: v1,
		},
		"an upgrade and no previous": {
			Name: "example.com/notifier", Version: 2, Schema: v2, Upgrade: upgradeV1,
		},
	} {
		t.Run(name, func(t *testing.T) {
			var cfg map[string]any
			err := config.LoadKind(versions+"v2.yaml", kind, &cfg)
			if err == nil {
				t.Fatal("accepted")
			}
			if !strings.Contains(err.Error(), "the binary declares") {
				t.Errorf("the refusal does not say whose mistake it is:\n%v", err)
			}
		})
	}
}
