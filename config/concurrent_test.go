package config_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/truvity/policy/config"
)

// Many goroutines load and validate against the same schema, and against a
// schema that does not compile, at once. Run with -race: the schema is
// compiled once and shared, and the answers must be the single-caller ones.
func TestConcurrentCallersShareOneCompilation(t *testing.T) {
	good := schema(t)
	bad := []byte(`{"type": `)

	var want shortener
	if err := config.Load("testdata/valid.yaml", good, &want); err != nil {
		t.Fatal(err)
	}
	wantBad := config.Validate(map[string]any{}, bad)
	if wantBad == nil {
		t.Fatal("a schema that is not JSON was accepted")
	}

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			var got shortener
			if err := config.Load("testdata/valid.yaml", good, &got); err != nil {
				t.Errorf("goroutine %d: a valid file was refused: %v", i, err)
				return
			}
			if got != want {
				t.Errorf("goroutine %d: decoded %+v, want %+v", i, got, want)
			}
			err := config.Load("testdata/unknown-key.yaml", good, &got)
			if err == nil || !strings.Contains(err.Error(), "not a key") && !strings.Contains(err.Error(), "additional") {
				t.Errorf("goroutine %d: unknown key not refused as before: %v", i, err)
			}
			if err := config.Validate(map[string]any{}, bad); err == nil || err.Error() != wantBad.Error() {
				t.Errorf("goroutine %d: bad schema error %v, want %v", i, err, wantBad)
			}
		}(i)
	}
	wg.Wait()
}

// The same schema text gives the same compiled result however many times it
// is asked, and a refusal's text does not change between calls.
func TestRepeatedValidationGivesTheSameAnswer(t *testing.T) {
	s := schema(t)
	first := config.Validate(map[string]any{"nope": 1}, s)
	if first == nil {
		t.Fatal("an invalid document was accepted")
	}
	for i := 0; i < 3; i++ {
		if again := config.Validate(map[string]any{"nope": 1}, s); again == nil || again.Error() != first.Error() {
			t.Fatalf("call %d: %v, want %v", i, again, first)
		}
	}
}
