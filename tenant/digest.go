package tenant

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/truvity/ocictl/pkg/static"
)

// Digest is the pin of a bundle: the digest of the OCI manifest that
// `ocictl static push` publishes for it, spelled "sha256:" and 64 lower case
// hex digits, the form a registry reports for a manifest.
//
// It is the digest `ocictl digest <dir>` prints on its "digest" line (the
// "manifest" line of its text output) for the directory the bundle is read
// from, with the options a tenant bundle is published with: the default
// artifact shape (artifactType application/vnd.ocictl.static.v1, an empty
// config, one tar+gzip layer) and nothing added to the manifest. The packing
// is ocictl's [static.Pack], so this package keeps no archive format of its
// own. A bundle published with `--shape image`, with any `--annotation` or
// named annotation flag, or with a source date (`--created`, or the
// SOURCE_DATE_EPOCH environment variable) is another manifest and has another
// digest, which is not the pin.
//
// The digest does not depend on the checkout: file modes, modification times,
// the order files were created in and the operating system do not reach it;
// the content and the path of each file do. A file that is not a regular file
// (a link, a device) is refused, because it would make two checkouts of one
// commit differ, and so is a bundle with no file.
//
// ocictl's packer reads a directory, so the bundle is copied into a temporary
// one first. The ocictl version this module requires fixes the bytes; moving
// it can move every pin, and the golden digests in the tests say so.
func Digest(fsys fs.FS) (string, error) {
	dir, err := os.MkdirTemp("", "tenant-bundle-")
	if err != nil {
		return "", fmt.Errorf("tenant: %w", err)
	}

	defer os.RemoveAll(dir) //nolint:errcheck // a leftover temporary directory is not a failed digest

	if err := materialise(fsys, dir); err != nil {
		return "", err
	}

	packed, err := static.Pack(dir)
	if err != nil {
		return "", fmt.Errorf("tenant: %w", err)
	}

	art, err := static.Artifact(packed, static.Options{Shape: static.ShapeArtifact})
	if err != nil {
		return "", fmt.Errorf("tenant: %w", err)
	}

	d, err := static.Describe(art)
	if err != nil {
		return "", fmt.Errorf("tenant: %w", err)
	}

	return d.Manifest, nil
}

// materialise writes the regular files of fsys below dir. Two paths that
// differ only in case are refused on every system, not only where the file
// system folds case: "A.yaml" and "a.yaml" are one file on a case-insensitive
// file system and two on Linux, and the digest must not depend on which. The
// check covers directories too, since "A/x" and "a/y" share a directory there.
// A name that is already on disk is an error, not an overwrite, as a backstop.
func materialise(fsys fs.FS, dir string) error {
	folded := map[string]string{}

	return fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		key := strings.ToLower(p)
		if prev, ok := folded[key]; ok && prev != p {
			return fmt.Errorf("tenant: %s and %s differ only in case", prev, p)
		}

		folded[key] = p

		if d.IsDir() {
			return nil
		}

		if !d.Type().IsRegular() {
			return fmt.Errorf("tenant: %s is not a regular file", p)
		}

		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}

		dst := filepath.Join(dir, filepath.FromSlash(p))

		if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
			return fmt.Errorf("tenant: %s: %w", p, err)
		}

		f, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) //nolint:gosec // dst is below a fresh temporary directory
		if err != nil {
			if errors.Is(err, fs.ErrExist) {
				return fmt.Errorf("tenant: %s: another file of the same name, differing only in case, is in the bundle", p)
			}

			return fmt.Errorf("tenant: %s: %w", p, err)
		}

		if _, err := f.Write(data); err != nil {
			_ = f.Close()

			return fmt.Errorf("tenant: %s: %w", p, err)
		}

		return f.Close()
	})
}
