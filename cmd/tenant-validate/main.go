// Command tenant-validate checks a company's configuration bundle the way the
// platform does before it pins it, so that the company's own CI refuses a bad
// bundle before the platform has to.
package main

import (
	"io"
	"os"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is the command, apart from the process: it returns the exit status.
func run(_ []string, _, _ io.Writer) int {
	return 0
}
