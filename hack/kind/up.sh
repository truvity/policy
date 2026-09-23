#!/usr/bin/env bash
# Stands up the local cluster the charts and the example are tested against:
# the same operators production runs, and a stand-in only where there is no
# operator to prove.
#
# Idempotent. Running it against an existing cluster upgrades in place, which
# is what makes it usable as a development loop rather than only as a CI step.
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

step "Gateway API ${GATEWAY_API_VERSION}"
# CRDs only. A route needs them to exist; nothing here needs a controller
# acting on one, and installing a gateway implementation would be minutes
# spent proving somebody else's software.
kubectl apply -f \
  "https://github.com/kubernetes-sigs/gateway-api/releases/download/${GATEWAY_API_VERSION}/standard-install.yaml"

step "CloudNativePG ${CNPG_CHART_VERSION}"
helm repo add cnpg https://cloudnative-pg.github.io/charts >/dev/null 2>&1 || true
helm repo add nats https://nats-io.github.io/k8s/helm/charts/ >/dev/null 2>&1 || true
helm repo update >/dev/null
helm upgrade --install cnpg cnpg/cloudnative-pg \
  --version "$CNPG_CHART_VERSION" \
  --namespace cnpg-system --create-namespace \
  --wait --timeout 5m

step "NATS ${NATS_CHART_VERSION} with JetStream"
# The memory store is enabled EXPLICITLY, and it is not a detail. With
# JetStream on and no memory store configured, the server accepts the
# controller's connection, reports healthy, and refuses every memory stream
# with "insufficient memory resources available" — so a chart that asks for
# one fails in the box and works in production, which is the opposite of
# what a test environment is for. The smoke test is what found it.
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

step "NACK ${NACK_CHART_VERSION}"
# The controller that turns a stream resource into a stream. Without it a
# chart's stream renders, applies, and nothing happens — which is exactly the
# failure a cluster is supposed to catch and a container cannot.
helm upgrade --install nack nats/nack \
  --version "$NACK_CHART_VERSION" \
  --namespace nats \
  --set jetstream.enabled=true \
  --set jetstream.nats.url=nats://nats.nats.svc:4222 \
  --wait --timeout 5m

step "cert-manager ${CERT_MANAGER_CHART_VERSION}"
# The authority, and the thing that mounts an identity into a pod. Installed
# before the driver, because the driver's approver is a cert-manager
# extension and the custom resources have to exist first.
helm repo add jetstack https://charts.jetstack.io >/dev/null 2>&1 || true
#
# THE APPROVER IS TURNED OFF HERE, AND THIS IS THE WHOLE POINT.
#
# cert-manager ships an approver that approves every request for an issuer it
# knows about. Leave it on and the identity driver's own approver never gets
# a say: an account that may create a request gets ANY identity it asks for,
# including its neighbour's. The driver still works, the certificates still
# mount, every log line still says success — and the attestation is
# decoration.
#
# Measured in this box before it was disabled: an account called `alice`
# submitted a request naming `bob` by hand and was issued a certificate for
# it. Nothing anywhere reported a problem.
helm upgrade --install cert-manager jetstack/cert-manager \
  --version "$CERT_MANAGER_CHART_VERSION" \
  --namespace cert-manager --create-namespace \
  --set crds.enabled=true \
  --set "extraArgs={--controllers=*\,-certificaterequests-approver}" \
  --wait --timeout 5m

step "the trust domain"
kubectl apply -f identity.yaml
# The authority signs from a secret cert-manager writes, so the issuer is not
# usable the moment it is applied. Waiting here rather than in the driver's
# install turns "certificate not ready" into a message about the authority.
kubectl -n cert-manager wait certificate/policy-trust --for=condition=Ready --timeout=2m

step "the identity driver ${CSI_DRIVER_SPIFFE_CHART_VERSION}"
# THE PROPERTY THIS PROVES: the driver asks for a certificate using the POD'S
# OWN account token, which the kubelet hands it, and the approver refuses any
# request whose identity is not the one the requester holds. A pod cannot ask
# for a certificate naming its neighbour's account — which is the difference
# between an identity and a claim.
helm upgrade --install csi-driver-spiffe jetstack/cert-manager-csi-driver-spiffe \
  --version "$CSI_DRIVER_SPIFFE_CHART_VERSION" \
  --namespace cert-manager \
  --set "app.trustDomain=${TRUST_DOMAIN}" \
  --set app.issuer.name=policy-workload \
  --set app.issuer.kind=ClusterIssuer \
  --set app.issuer.group=cert-manager.io \
  --set app.driver.volumes[0].name=root-cas \
  --set app.driver.volumes[0].secret.secretName=policy-trust \
  --set app.driver.volumeMounts[0].name=root-cas \
  --set app.driver.volumeMounts[0].mountPath=/var/run/secrets/cert-manager-csi-driver-spiffe \
  --set app.driver.sourceCABundle=/var/run/secrets/cert-manager-csi-driver-spiffe/ca.crt \
  --wait --timeout 5m

step "S3"
sed "s|LOCALSTACK_IMAGE_PLACEHOLDER|${LOCALSTACK_IMAGE}|" localstack.yaml | kubectl apply -f -
kubectl -n object-store rollout status deployment/s3 --timeout=5m

step "what the box can be asked for"
./verify.sh

printf '\n\033[1mready in %ds\033[0m — kubectl context kind-%s\n' "$((SECONDS - started))" "$CLUSTER"
