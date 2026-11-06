package tenant_test

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/truvity/policy/tenant"
)

func files(kv ...string) fstest.MapFS {
	m := fstest.MapFS{}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = &fstest.MapFile{Data: []byte(kv[i+1]), Mode: 0o644}
	}

	return m
}

func digestOf(t *testing.T, fsys fs.FS) string {
	t.Helper()

	d, err := tenant.Digest(fsys)
	if err != nil {
		t.Fatal(err)
	}

	return d
}

// The same content gives the same digest whatever the checkout did to it:
// file modes, modification times and the order the files were created in.
func TestDigestIsStableUnderModesTimesAndOrder(t *testing.T) {
	a := fstest.MapFS{
		"tenant.yaml":   &fstest.MapFile{Data: []byte("a"), Mode: 0o644, ModTime: time.Unix(1, 0)},
		"projects.yaml": &fstest.MapFile{Data: []byte("b"), Mode: 0o600, ModTime: time.Unix(2, 0)},
		"x/y.yaml":      &fstest.MapFile{Data: []byte("c"), Mode: 0o755, ModTime: time.Unix(3, 0)},
	}
	b := fstest.MapFS{
		"x/y.yaml":      &fstest.MapFile{Data: []byte("c"), Mode: 0o644, ModTime: time.Unix(9, 9)},
		"projects.yaml": &fstest.MapFile{Data: []byte("b"), Mode: 0o644},
		"tenant.yaml":   &fstest.MapFile{Data: []byte("a")},
	}

	if da, db := digestOf(t, a), digestOf(t, b); da != db {
		t.Errorf("modes, times or order changed the digest: %s != %s", da, db)
	}
}

func TestDigestChangesWithContentAndPath(t *testing.T) {
	base := digestOf(t, files("tenant.yaml", "a", "projects.yaml", "b"))

	for name, other := range map[string]fstest.MapFS{
		"content":       files("tenant.yaml", "a", "projects.yaml", "c"),
		"a path":        files("tenant.yaml", "a", "project.yaml", "b"),
		"a file less":   files("tenant.yaml", "a"),
		"a file more":   files("tenant.yaml", "a", "projects.yaml", "b", "ci.yaml", ""),
		"swapped files": files("tenant.yaml", "b", "projects.yaml", "a"),
	} {
		if got := digestOf(t, other); got == base {
			t.Errorf("%s did not change the digest", name)
		}
	}
}

// A link makes two checkouts of one commit differ, so it is not a file.
func TestDigestRefusesWhatIsNotARegularFile(t *testing.T) {
	m := files("tenant.yaml", "a")
	m["link.yaml"] = &fstest.MapFile{Data: []byte("tenant.yaml"), Mode: fs.ModeSymlink | 0o777}

	if _, err := tenant.Digest(m); err == nil {
		t.Fatal("a symlink was digested")
	}
}

// Two names that differ only in case would be one file on a case-insensitive
// file system and two on Linux, so the digest would depend on the machine.
// Digest refuses both the file and the directory form on every system.
func TestDigestRefusesNamesThatDifferOnlyInCase(t *testing.T) {
	for name, m := range map[string]fstest.MapFS{
		"files":       files("tenant.yaml", "x", "A.yaml", "1", "a.yaml", "2"),
		"directories": files("tenant.yaml", "x", "A/x", "1", "a/y", "2"),
	} {
		if _, err := tenant.Digest(m); err == nil {
			t.Errorf("%s: a bundle with names differing only in case was digested", name)
		}
	}
}

// A bundle with no file has no artifact to pin: the packer refuses it, and so
// does Digest.
func TestDigestRefusesAnEmptyBundle(t *testing.T) {
	if _, err := tenant.Digest(fstest.MapFS{}); err == nil {
		t.Fatal("an empty bundle was digested")
	}
}

// The digest has a spelling other tools can read: "sha256:" and 64 hex digits,
// the form a registry reports for a manifest.
func TestDigestIsSpelledLikeAContentDigest(t *testing.T) {
	d := digestOf(t, files("tenant.yaml", "a"))
	if len(d) != len("sha256:")+64 || d[:7] != "sha256:" {
		t.Errorf("%q", d)
	}
}

// The pin is the manifest digest ocictl produces, so it moves when the ocictl
// this module requires changes how it packs. These constants are what
// `go tool ocictl digest <dir> --output json` prints (its "digest" field) for
// the directory each case reads (TestDigestEqualsWhatOcictlPrints runs the
// tool on both), at the version in go.mod. A change here is
// a change of interface: every pin made before it stops matching. Move them
// only in the change that moves ocictl, and say so in the changelog.
func TestDigestIsPinned(t *testing.T) {
	for name, c := range map[string]struct {
		fsys fs.FS
		want string
	}{
		"the smallest bundle": {
			fsys: os.DirFS(validManifestOnly),
			want: "sha256:388cd55bf97a66c1598b0099c77b43a0e4a5561537a031d3a5a33f8d92f08069",
		},
		"the full fixture": {
			fsys: os.DirFS(validFull),
			want: "sha256:c4888d959cbd8bdccec6c586e8e717ce456ac2bb454eb80006e136ba5a796fde",
		},
	} {
		if got := digestOf(t, c.fsys); got != c.want {
			t.Errorf("%s: got %s, want %s", name, got, c.want)
		}
	}
}

// ocictlDigest runs `ocictl digest <dir> --output json` through the ocictl
// this module pins (`go tool`) and returns its "digest" field, with the one
// environment variable that would add a creation time to the manifest taken
// out: a pin is made without a source date.
func ocictlDigest(t *testing.T, dir string) string {
	t.Helper()

	cmd := exec.Command("go", "tool", "ocictl", "digest", dir, "--output", "json")
	cmd.Env = withoutEnv(os.Environ(), "SOURCE_DATE_EPOCH")

	out, err := cmd.Output()
	if err != nil {
		stderr := ""

		var ee *exec.ExitError
		if errors.As(err, &ee) {
			stderr = string(ee.Stderr)
		}

		t.Fatalf("ocictl digest %s: %v\n%s", dir, err, stderr)
	}

	var res struct {
		Digest string `json:"digest"`
	}

	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("ocictl digest printed %q: %v", out, err)
	}

	if res.Digest == "" {
		t.Fatalf("ocictl digest printed no digest: %q", out)
	}

	return res.Digest
}

func withoutEnv(env []string, name string) []string {
	var out []string

	for _, kv := range env {
		if !strings.HasPrefix(kv, name+"=") {
			out = append(out, kv)
		}
	}

	return out
}

// The point of the pin: Digest is the digest `ocictl digest` prints for the
// same files, which is the digest `ocictl static push` gives the manifest it
// publishes. A company's CI checks with the tool and the platform computes it
// with this function; they must be one value.
func TestDigestEqualsWhatOcictlPrints(t *testing.T) {
	// A bundle on disk, read in place: the full fixture and the smallest one,
	// the two directories whose digests TestDigestIsPinned holds.
	for _, dir := range []string{validFull, validManifestOnly} {
		if got, want := digestOf(t, os.DirFS(dir)), ocictlDigest(t, dir); got != want {
			t.Errorf("%s: Digest %s, ocictl digest %s", dir, got, want)
		}
	}

	// A bundle that exists only in memory, with a nested directory and a name
	// that sorts differently by whole path than by directory.
	mem := files("tenant.yaml", "schema: tenant.v1\n", "a/b", "1", "a.x", "2", "c", "3")

	dir := t.TempDir()

	for name, f := range mem {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(p, f.Data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if got, want := digestOf(t, mem), ocictlDigest(t, dir); got != want {
		t.Errorf("in-memory bundle: Digest %s, ocictl digest %s", got, want)
	}
}

// A source date is another manifest, so a pin made with one is not the digest
// of the bundle. The contract tells a publisher to leave it out; this test
// keeps that sentence true.
func TestASourceDateMakesAnotherManifest(t *testing.T) {
	cmd := exec.Command("go", "tool", "ocictl", "digest", validFull, "--output", "json")
	cmd.Env = append(withoutEnv(os.Environ(), "SOURCE_DATE_EPOCH"), "SOURCE_DATE_EPOCH=1700000000")

	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}

	var res struct {
		Digest string `json:"digest"`
	}

	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatal(err)
	}

	if res.Digest == digestOf(t, os.DirFS(validFull)) {
		t.Error("a source date did not change the manifest digest: the contract's warning is stale")
	}
}
