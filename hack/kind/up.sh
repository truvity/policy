#!/usr/bin/env bash
# Stands up the local cluster the charts and every example are tested
# against: SERVERS ONLY, nothing that binds a chart to one platform's
# choices.
#
# What it carries and why: hack/kind/README.md. What that split means for
# an example's own database, stream and bucket: each example brings its own
# FIXTURE, under its own directory, naming what it needs by the names its
# chart takes — see examples/url-shortener/e2e/fixture.
#
# Idempotent. Running it against an existing cluster upgrades in place, which
# is what makes it usable as a development loop rather than only as a CI
# step.
set -euo pipefail
cd "$(dirname "$0")"
# shellcheck disable=SC1091
source versions.env

started=$SECONDS
step() { printf '\n\033[1m==> %s\033[0m\n' "$1"; }

step "cluster ${CLUSTER}"
if kind get clusters 2>/dev/null | grep -qx "$CLUSTER"; then
  echo "already exists"
else
  kind create cluster --name "$CLUSTER" --config cluster.yaml --image "$NODE_IMAGE" --wait 120s
fi
kubectl config use-context "kind-${CLUSTER}" >/dev/null

step "the local registry ${REGISTRY_NAME}"
# kind's own documented recipe: https://kind.sigs.k8s.io/docs/user/local-registry/
#
# A registry rather than `kind load`, because that is what a chart's own
# `images.*.registry` value expects to reach, and because a second cluster
# reusing this box (the whole point of making it example-agnostic) may need
# to push its own images without a rebuild loading them straight into nodes.
if [ "$(docker inspect -f '{{.State.Running}}' "$REGISTRY_NAME" 2>/dev/null || true)" != true ]; then
  docker run -d --restart=always -p "127.0.0.1:${REGISTRY_PORT}:5000" \
    --network bridge --name "$REGISTRY_NAME" "$REGISTRY_IMAGE" >/dev/null
fi

# containerd on every node is told (by cluster.yaml's containerdConfigPatches)
# to read a per-registry directory for `localhost:${REGISTRY_PORT}`; this
# writes the file that directory needs. Redone on every run rather than
# guarded, because it is one write and idempotent on its own.
registry_dir="/etc/containerd/certs.d/localhost:${REGISTRY_PORT}"
for node in $(kind get nodes --name "$CLUSTER"); do
  docker exec "$node" mkdir -p "$registry_dir"
  docker exec -i "$node" cp /dev/stdin "${registry_dir}/hosts.toml" <<EOF
[host."http://${REGISTRY_NAME}:5000"]
EOF
done

# The registry and the cluster's nodes must be on the same docker network for
# the host name above to resolve. kind names its network "kind"; connecting
# is a no-op if it is already connected.
if [ "$(docker inspect -f '{{json .NetworkSettings.Networks.kind}}' "$REGISTRY_NAME")" = null ]; then
  docker network connect kind "$REGISTRY_NAME"
fi

# Documents the mapping for anything that reads it — kind's own convention,
# https://github.com/kubernetes/enhancements/tree/master/keps/sig-cluster-lifecycle/generic/1755-communicating-a-local-registry
kubectl apply -f - >/dev/null <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:${REGISTRY_PORT}"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF

step "Postgres"
kubectl create namespace postgres --dry-run=client -o yaml | kubectl apply -f - >/dev/null

# The superuser's password, generated once per cluster and never again: a
# `kubectl create` that finds the Secret already there is left alone, so
# re-running the box does not rotate a credential every already-provisioned
# database still uses. Created BEFORE the Deployment below, which mounts it
# — applying the Deployment first would still converge once the Secret
# exists, but only after a pod sat failing for no reason a reader of the log
# could guess.
if ! kubectl -n postgres get secret postgres-superuser >/dev/null 2>&1; then
  kubectl -n postgres create secret generic postgres-superuser \
    --type=kubernetes.io/basic-auth \
    --from-literal=username=postgres \
    --from-literal=password="$(head -c 24 /dev/urandom | base64 | tr -d '/+=')"
fi

# A self-signed server certificate, so `sslmode=require` — which every
# chart's rendered connection string asks for — has something to negotiate.
# `sslmode=require` only asks for an encrypted channel; it does not verify
# the certificate against any authority, so a throwaway self-signed pair is
# the whole answer here. Idempotent on the same terms as the password above.
if ! kubectl -n postgres get secret postgres-tls >/dev/null 2>&1; then
  tls=$(mktemp -d)
  trap 'rm -rf "$tls"' EXIT
  openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:prime256v1 \
    -keyout "$tls/tls.key" -out "$tls/tls.crt" \
    -days 30 -nodes -subj "/CN=postgres" >/dev/null 2>&1
  kubectl -n postgres create secret tls postgres-tls \
    --cert="$tls/tls.crt" --key="$tls/tls.key"
fi

sed "s|POSTGRES_IMAGE_PLACEHOLDER|${POSTGRES_IMAGE}|g" postgres.yaml | kubectl apply -f -
kubectl -n postgres rollout status deployment/postgres --timeout=5m

step "NATS ${NATS_CHART_VERSION} with JetStream"
# The memory store is enabled EXPLICITLY, and it is not a detail. With
# JetStream on and no memory store configured, the server accepts a
# connection, reports healthy, and refuses every memory stream with
# "insufficient memory resources available" — so a chart that asks for one
# fails in the box and works in production, which is the opposite of what a
# test environment is for. The smoke test is what found it.
helm repo add nats https://nats-io.github.io/k8s/helm/charts/ >/dev/null 2>&1 || true
helm repo update nats >/dev/null
helm upgrade --install nats nats/nats \
  --version "$NATS_CHART_VERSION" \
  --namespace nats --create-namespace \
  --set config.jetstream.enabled=true \
  --set config.jetstream.memoryStore.enabled=true \
  --set config.jetstream.memoryStore.maxSize=256Mi \
  --wait --timeout 5m

# JetStream's store limits are read at start-up and are NOT hot-reloadable.
# An upgrade that changes them rewrites the config map, the reloader picks it
# up, the server reports healthy — and still refuses every memory stream,
# because the limit it is running with is the one it booted with. A restart
# of one pod costs seconds; a box that is silently unusable costs an
# afternoon.
kubectl -n nats rollout restart statefulset/nats
kubectl -n nats rollout status statefulset/nats --timeout=5m

step "S3"
sed "s|LOCALSTACK_IMAGE_PLACEHOLDER|${LOCALSTACK_IMAGE}|" localstack.yaml | kubectl apply -f -
kubectl -n object-store rollout status deployment/s3 --timeout=5m

step "what the box can be asked for"
./verify.sh

printf '\n\033[1mready in %ds\033[0m — kubectl context kind-%s\n' "$((SECONDS - started))" "$CLUSTER"
