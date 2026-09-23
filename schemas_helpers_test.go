package policy_test

import (
	"encoding/json"
	"errors"

	"github.com/truvity/policy/config"
)

func jsonUnmarshal(b []byte, v any) error { return json.Unmarshal(b, v) }

func asConfigError(err error, target **config.Error) bool { return errors.As(err, target) }
