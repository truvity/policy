// Code generated from Pkl module `spike.contract.Showcase`. DO NOT EDIT.
package showcase

import (
	"context"

	"github.com/apple/pkl-go/pkl"
	"spike.invalid/pklconf/gen/vocab/cspmode"
	"spike.invalid/pklconf/gen/vocab/loglevel"
	"spike.invalid/pklconf/gen/vocab/otelprotocol"
	"spike.invalid/pklconf/gen/vocab/pullpolicy"
	"spike.invalid/pklconf/gen/vocab/tlsmode"
)

// One optional property per vocabulary type, so that every alias can be
// probed on its own: this is the module the generated probe fixtures run
// against, and what proves each alias's constraint and annotation agree.
type Showcase struct {
	Port *int `pkl:"port"`

	Percent *int `pkl:"percent"`

	Ratio *float64 `pkl:"ratio"`

	PositiveInt *int `pkl:"positiveInt"`

	NonNegativeInt *int `pkl:"nonNegativeInt"`

	Bytes *string `pkl:"bytes"`

	Duration *string `pkl:"duration"`

	NonEmptyString *string `pkl:"nonEmptyString"`

	Url *string `pkl:"url"`

	Hostname *string `pkl:"hostname"`

	HostPort *string `pkl:"hostPort"`

	PostgresUrl *string `pkl:"postgresUrl"`

	ImageDigest *string `pkl:"imageDigest"`

	AbsPath *string `pkl:"absPath"`

	RootedPath *string `pkl:"rootedPath"`

	EnvName *string `pkl:"envName"`

	HttpOrigin *string `pkl:"httpOrigin"`

	ReportUri *string `pkl:"reportUri"`

	CollectorUrl *string `pkl:"collectorUrl"`

	PublicKey *string `pkl:"publicKey"`

	UrlPath *string `pkl:"urlPath"`

	UrlPathOrEmpty *string `pkl:"urlPathOrEmpty"`

	DnsName *string `pkl:"dnsName"`

	LogLevel *loglevel.LogLevel `pkl:"logLevel"`

	TlsMode *tlsmode.TlsMode `pkl:"tlsMode"`

	CspMode *cspmode.CspMode `pkl:"cspMode"`

	PullPolicy *pullpolicy.PullPolicy `pkl:"pullPolicy"`

	OtelProtocol *otelprotocol.OtelProtocol `pkl:"otelProtocol"`
}

// LoadFromPath loads the pkl module at the given path and evaluates it into a Showcase
func LoadFromPath(ctx context.Context, path string) (ret Showcase, err error) {
	evaluator, err := pkl.NewEvaluator(ctx, pkl.PreconfiguredOptions)
	if err != nil {
		return ret, err
	}
	defer func() {
		cerr := evaluator.Close()
		if err == nil {
			err = cerr
		}
	}()
	ret, err = Load(ctx, evaluator, pkl.FileSource(path))
	return ret, err
}

// Load loads the pkl module at the given source and evaluates it with the given evaluator into a Showcase
func Load(ctx context.Context, evaluator pkl.Evaluator, source *pkl.ModuleSource) (Showcase, error) {
	var ret Showcase
	err := evaluator.EvaluateModule(ctx, source, &ret)
	return ret, err
}
