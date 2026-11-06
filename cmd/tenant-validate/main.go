// Command tenant-validate checks a company's configuration bundle the way the
// platform does before it pins it, so that the company's own CI refuses a bad
// bundle before the platform has to.
//
//	tenant-validate [-digest] DIR
//
// DIR is the bundle: the directory holding tenant.yaml and the section files
// (docs/contracts/tenant.md). The command runs [tenant.Load] and
// [tenant.Bundle.Validate], which need nothing but the files, and then
// [tenant.Digest]. It does not run [tenant.Bundle.Check]: that compares the
// bundle with the platform's record of the company (zones, e-mail domains,
// registries, ceilings), which a company's repository does not hold and the
// platform alone supplies, so the platform runs it when it takes the bundle.
//
// A bundle that passes prints the company and the pin to standard output:
//
//	acme sha256:...
//
// and with -digest only the pin. The pin is the digest `ocictl digest DIR`
// prints and a registry reports for the manifest `ocictl static push DIR`
// publishes. A bundle that fails prints every failure to standard error, one
// per line, each with its stable code, and nothing to standard output.
//
// The exit status is 0 for an accepted bundle, 1 for a refused one, and 2 when
// the check could not run (a bad flag, no directory, a path that cannot be
// read), so that a script can tell a bad bundle from a bad invocation.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/truvity/policy/tenant"
)

const (
	statusAccepted = 0
	statusRefused  = 1
	statusUsage    = 2
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is the command, apart from the process: it returns the exit status.
func run(args []string, stdout, stderr io.Writer) int {
	fl := flag.NewFlagSet("tenant-validate", flag.ContinueOnError)
	fl.SetOutput(stderr)
	fl.Usage = func() {
		say(stderr, "usage: tenant-validate [-digest] DIR")
		say(stderr, "  Check the company bundle in DIR and print its pin.")
		fl.PrintDefaults()
	}

	digestOnly := fl.Bool("digest", false, "print only the pin (the OCI manifest digest)")

	if err := fl.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return statusAccepted
		}

		return statusUsage
	}

	if fl.NArg() != 1 {
		fl.Usage()

		return statusUsage
	}

	dir := fl.Arg(0)

	info, err := os.Stat(dir)
	if err != nil {
		say(stderr, "tenant-validate: %v", err)

		return statusUsage
	}

	if !info.IsDir() {
		say(stderr, "tenant-validate: %s is not a directory", dir)

		return statusUsage
	}

	fsys := os.DirFS(dir)

	bundle, err := tenant.Load(fsys)
	if err == nil {
		err = bundle.Validate()
	}

	if err != nil {
		return refused(stderr, err)
	}

	pin, err := tenant.Digest(fsys)
	if err != nil {
		return refused(stderr, err)
	}

	if *digestOnly {
		say(stdout, "%s", pin)
	} else {
		say(stdout, "%s %s", bundle.Manifest.Company, pin)
	}

	return statusAccepted
}

// refused prints why the bundle was refused (every failure of
// [tenant.Errors], or why it cannot be packed) and returns its status.
func refused(stderr io.Writer, err error) int {
	say(stderr, "%s", err)

	return statusRefused
}

// say writes one line. A command that cannot write to its own output has
// nowhere to report it, so the error is dropped.
func say(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format+"\n", args...)
}
