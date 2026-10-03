// Code generated from Pkl module `spike.contract.Platform`. DO NOT EDIT.
package platform

import (
	"context"

	"github.com/apple/pkl-go/pkl"
	"spike.invalid/pklconf/gen/vocab/pullpolicy"
)

// What the platform provides one component of a service chart: the image, the replicas, the account it runs as, how it is probed, where its identity is mounted, which secrets reach it as environment variables, and how it exports telemetry. The shape of the `platform` block of a chart's values, read by the library chart (decision 0009). Nothing here is the service's own configuration: that is the chart's `config` block, which is the service's schema and nothing else.
type Platform struct {
	// Where the image is. A digest when there is one; a tag only when there is not. Left out, the library reads the chart's own top-level `images.<component>`, the map a release stamps and refuses to publish with an empty digest.
	Image *PlatformImage `pkl:"image"`

	// Defaults to IfNotPresent.
	ImagePullPolicy *pullpolicy.PullPolicy `pkl:"imagePullPolicy"`

	// Defaults to 1. A service that must survive a rollout runs more than one: one instance cannot be replaced without a gap whatever the strategy says.
	Replicas *int `pkl:"replicas"`

	// The rolling update. The defaults make a rollout gapless: the replacement is READY before the incumbent is touched.
	Strategy *PlatformStrategy `pkl:"strategy"`

	// Kubernetes' own resource requirements. Open: it is passed through unchanged.
	Resources *pkl.Object `pkl:"resources"`

	// Who the process is. Every field defaults to 65532, the unprivileged user the runtime images run as; `fsGroup` is the one that matters, because a CSI driver writes what it mounts owned by root.
	PodSecurity *PlatformPodSecurity `pkl:"podSecurity"`

	// The account this component runs as. EVERY component has its own, always; `default` is refused. The library grants nothing: annotations are where a platform binds the account to rights outside the cluster, and which mechanism does that is the platform's.
	ServiceAccount *PlatformServiceAccount `pkl:"serviceAccount"`

	// A component that listens has a Service on the ports of its own `config.listen`; one that does not has none.
	Service *PlatformService `pkl:"service"`

	// How the component is probed, on the listener its own `config.probes` binds. Liveness is nothing but the process: a probe that checks a dependency restarts a healthy process and makes an outage worse.
	Probes *PlatformProbes `pkl:"probes"`

	// The service's own shutdown budget is `config.drain.seconds`; the library derives the grace period from it and this delay, so the three numbers cannot disagree.
	Drain *PlatformDrain `pkl:"drain"`

	// Where the platform mounts the workload identity. The files the component's own `config.tls` names must be under `mountPath`; the render refuses a file that is not.
	Tls *PlatformTls `pkl:"tls"`

	// OpenTelemetry's own environment variables (decision 0006): they leave the chart as variables, never as configuration keys. No endpoint means do not export.
	Telemetry *PlatformTelemetry `pkl:"telemetry"`

	// The environment variables that carry SECRETS, and nothing else (decision 0002): variable name to the Secret and key its value comes from. The configuration file names the VARIABLE; the value never appears in a values file or a render.
	Secrets *map[string]Secret `pkl:"secrets"`

	// Environment a platform CLIENT LIBRARY reads (a database client's connection variables, for example), never the service's own configuration: a service takes no other structural input than its file (decision 0002). A secret does not belong here; declare it in `secrets`.
	Env *[]EnvVar `pkl:"env"`

	// Extra pod volumes, in Kubernetes' own shape, for what a client library mounts (a trust bundle, a password file). Open: passed through unchanged.
	Volumes *[]pkl.Object `pkl:"volumes"`

	// The mounts for `volumes`, in Kubernetes' own shape.
	VolumeMounts *[]pkl.Object `pkl:"volumeMounts"`

	// How the file reaches the process. The path is ONE argument or ONE environment variable, never both (decision 0002).
	Config *PlatformConfig `pkl:"config"`

	ConfigMap *PlatformConfigMap `pkl:"configMap"`
}

// LoadFromPath loads the pkl module at the given path and evaluates it into a Platform
func LoadFromPath(ctx context.Context, path string) (ret Platform, err error) {
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

// Load loads the pkl module at the given source and evaluates it with the given evaluator into a Platform
func Load(ctx context.Context, evaluator pkl.Evaluator, source *pkl.ModuleSource) (Platform, error) {
	var ret Platform
	err := evaluator.EvaluateModule(ctx, source, &ret)
	return ret, err
}
