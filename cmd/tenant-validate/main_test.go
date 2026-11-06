package main

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/truvity/policy/tenant"
)

const (
	validFull         = "../../tenant/testdata/valid/full"
	validManifestOnly = "../../tenant/testdata/valid/manifest-only"
	refuseDir         = "../../tenant/testdata/refuse"
)

var digestForm = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// exec runs the command with args and returns its status and both streams.
func exec(t *testing.T, args ...string) (status int, stdout, stderr string) {
	t.Helper()

	var out, errOut bytes.Buffer

	status = run(args, &out, &errOut)

	return status, out.String(), errOut.String()
}

// digestOf is the pin the tenant package computes for a directory.
func digestOf(t *testing.T, dir string) string {
	t.Helper()

	d, err := tenant.Digest(os.DirFS(dir))
	if err != nil {
		t.Fatal(err)
	}

	return d
}

// bundleWith copies the full valid bundle into a new directory and lays the
// files of overlay over it: a valid bundle with one change, as the refusal
// fixtures are written.
func bundleWith(t *testing.T, overlay string) string {
	t.Helper()

	dst := t.TempDir()

	for _, src := range []string{validFull, overlay} {
		err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}

			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}

			return os.WriteFile(filepath.Join(dst, filepath.Base(p)), data, 0o600)
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	return dst
}

func TestAcceptedBundleIsReportedWithItsPin(t *testing.T) {
	for name, dir := range map[string]string{
		"the full bundle": validFull,
		"manifest only":   validManifestOnly,
	} {
		status, stdout, stderr := exec(t, dir)

		if status != 0 || stderr != "" {
			t.Fatalf("%s: status %d, stderr %q", name, status, stderr)
		}

		want := "acme " + digestOf(t, dir) + "\n"
		if stdout != want {
			t.Errorf("%s: stdout %q, want %q", name, stdout, want)
		}
	}
}

func TestDigestFlagPrintsOnlyThePin(t *testing.T) {
	status, stdout, stderr := exec(t, "-digest", validFull)

	if status != 0 || stderr != "" {
		t.Fatalf("status %d, stderr %q", status, stderr)
	}

	if !digestForm.MatchString(strings.TrimSuffix(stdout, "\n")) || stdout != digestOf(t, validFull)+"\n" {
		t.Errorf("stdout %q is not the pin", stdout)
	}
}

// A bundle that Load or Validate refuses is refused with status 1, every
// failure on its own line of stderr with its code, and nothing on stdout: a
// script that reads the pin must not get one for a bundle the platform would
// refuse.
func TestRefusedBundleIsRefusedWithItsCodes(t *testing.T) {
	for code, file := range map[tenant.Code]string{
		tenant.UnknownField:     "projects.yaml",
		tenant.Invalid:          "projects.yaml",
		tenant.UnknownInterface: "tenant.yaml",
		tenant.Duplicate:        "projects.yaml",
		tenant.MissingReason:    "access.yaml",
	} {
		dir := bundleWith(t, filepath.Join(refuseDir, string(code)))

		for _, args := range [][]string{{dir}, {"-digest", dir}} {
			status, stdout, stderr := exec(t, args...)

			if status != 1 {
				t.Errorf("%s %v: status %d, want 1", code, args, status)
			}

			if stdout != "" {
				t.Errorf("%s %v: stdout %q, want nothing", code, args, stdout)
			}

			if !strings.Contains(stderr, file) || !strings.Contains(stderr, "["+string(code)+"]") {
				t.Errorf("%s %v: stderr %q names neither %s nor the code", code, args, stderr, file)
			}
		}
	}
}

// Every failure of one pass is printed, not the first.
func TestEveryFailureIsPrinted(t *testing.T) {
	dir := bundleWith(t, filepath.Join(refuseDir, "duplicate"))

	if err := os.WriteFile(filepath.Join(dir, "access.yaml"), mustRead(t, filepath.Join(refuseDir, "missing-reason", "access.yaml")), 0o600); err != nil {
		t.Fatal(err)
	}

	_, _, stderr := exec(t, dir)

	for _, code := range []tenant.Code{tenant.Duplicate, tenant.MissingReason} {
		if !strings.Contains(stderr, "["+string(code)+"]") {
			t.Errorf("stderr %q lacks %s", stderr, code)
		}
	}

	if n := strings.Count(strings.TrimSuffix(stderr, "\n"), "\n") + 1; n < 2 {
		t.Errorf("stderr has %d lines, want one per failure", n)
	}
}

func mustRead(t *testing.T, p string) []byte {
	t.Helper()

	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}

	return data
}

// Not being able to run the check is not a refusal: status 2, so a script can
// tell a bad bundle from a bad invocation.
func TestUsageErrorsAreNotRefusals(t *testing.T) {
	file := filepath.Join(t.TempDir(), "afile")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	for name, args := range map[string][]string{
		"no directory":      {},
		"two directories":   {validFull, validManifestOnly},
		"an unknown flag":   {"-nope", validFull},
		"a missing path":    {filepath.Join(t.TempDir(), "absent")},
		"a file, not a dir": {file},
	} {
		status, stdout, stderr := exec(t, args...)

		if status != 2 || stdout != "" || stderr == "" {
			t.Errorf("%s: status %d, stdout %q, stderr %q; want 2, nothing, a message", name, status, stdout, stderr)
		}
	}
}

func TestHelpIsNotAFailure(t *testing.T) {
	status, stdout, stderr := exec(t, "-h")

	if status != 0 || stdout != "" || !strings.Contains(stderr, "tenant-validate") {
		t.Errorf("status %d, stdout %q, stderr %q", status, stdout, stderr)
	}
}
