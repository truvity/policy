package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// PathFrom returns the path of the binary's one configuration file
// (docs/contracts/service.md, rule 1): the argument `--config <path>` if the
// command line has it, otherwise the environment variable envName.
//
// args is the command line without the program's name, os.Args[1:]. The
// argument is spelled `--config` or `-config`, with the path as the next
// argument or after `=`: one dash is what Go's flag package and the library
// chart (charts/service-lib) write, two is what everything else does, and
// both are the same flag. Nothing after `--` is read as an option.
//
// The argument wins over the variable, so that a person can point a binary
// at another file without unsetting what the platform set. envName is the
// service's own, conventionally `<APP>_CONFIG`.
//
// Every other argument is left alone: a binary with subcommands or a flag
// parser of its own declares `config` there too, and this reads the same
// argument the parser does. What it refuses is the config argument with no
// value, the config argument given twice (a person reading the command line
// cannot tell which wins), an empty variable, and neither at all.
func PathFrom(args []string, envName string) (string, error) {
	path, found, err := configArgument(args)
	if err != nil {
		return "", &Error{Err: err}
	}
	if found {
		return path, nil
	}

	if envName == "" {
		return "", &Error{Err: errors.New("no configuration file: pass --config <path>")}
	}
	v, set := os.LookupEnv(envName)
	if !set {
		return "", &Error{Err: fmt.Errorf("no configuration file: pass --config <path> or set %s", envName)}
	}
	if v == "" {
		return "", &Error{Err: fmt.Errorf("no configuration file: %s is empty", envName)}
	}
	return v, nil
}

// configArgument finds the config argument in args.
func configArgument(args []string) (path string, found bool, err error) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			break
		}
		var value string
		switch {
		case a == "--config" || a == "-config":
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				value = args[i+1]
				i++
			}
		case strings.HasPrefix(a, "--config="):
			value = strings.TrimPrefix(a, "--config=")
		case strings.HasPrefix(a, "-config="):
			value = strings.TrimPrefix(a, "-config=")
		default:
			continue
		}
		if value == "" {
			return "", false, errors.New("--config has no value")
		}
		if found {
			return "", false, errors.New("--config is given more than once")
		}
		path, found = value, true
	}
	return path, found, nil
}
