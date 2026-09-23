#!/usr/bin/env bash
# Asserts the box offers what a chart's tests will ask of it.
#
# Separate from up.sh, and run by it, because the interesting question is not
# "did the installs report success" — helm --wait already answers that — but
# "is each thing usable". An operator whose CRD is missing, or a JetStream
# that is off, reports a healthy pod either way.
set -euo pipefail

fail=0
ok()   { printf '  ok    %s\n' "$1"; }
bad()  { printf '  MISSING %s\n' "$1"; fail=1; }

have_crd() { kubectl get crd "$1" >/dev/null 2>&1 && ok "$1" || bad "$1"; }

have_crd clusters.postgresql.cnpg.io
have_crd streams.jetstream.nats.io
have_crd consumers.jetstream.nats.io
have_crd gateways.gateway.networking.k8s.io
have_crd httproutes.gateway.networking.k8s.io

# JetStream on the server itself, not merely a controller that could talk to
# one: a stream resource applied against a server without JetStream is
# accepted and never becomes a stream.
if kubectl -n nats exec statefulset/nats -c nats -- nats-server --help >/dev/null 2>&1 ||
   kubectl -n nats get statefulset nats >/dev/null 2>&1; then
  ok "nats statefulset"
else
  bad "nats statefulset"
fi

# The limit the server is RUNNING with, not the one its config map holds.
# With JetStream on and no memory limit, the server is healthy, the
# controller connects, and every memory stream is refused with "insufficient
# memory resources available" — a chart that asks for one then fails in the
# box and works in production, which is the opposite of what a box is for.
max_memory=$(kubectl -n nats exec nats-0 -c nats -- wget -qO- http://127.0.0.1:8222/varz 2>/dev/null |
  sed -n 's/.*"max_memory": *\([0-9]*\).*/\1/p' | head -1)
if [ -n "${max_memory:-}" ] && [ "$max_memory" -gt 0 ] 2>/dev/null; then
  ok "jetstream memory store (${max_memory} bytes)"
else
  bad "jetstream memory store — the server is running with no memory limit"
fi

if kubectl -n object-store get endpoints s3 -o jsonpath='{.subsets[*].addresses[*].ip}' 2>/dev/null | grep -q .; then
  ok "s3 endpoint"
else
  bad "s3 endpoint"
fi

# THE CHECK THIS BOX EXISTS FOR, in its purest form.
#
# cert-manager ships an approver that approves every request for an issuer it
# knows. With it left on, the identity driver's own approver never gets a
# say, and any account that may create a request receives ANY identity it
# asks for — including its neighbour's. Everything still works: certificates
# mount, services connect, every log line says success. The attestation is
# decoration and nothing reports it.
#
# Measured here before it was disabled: an account called `alice` submitted a
# request naming `bob` by hand and was issued a certificate for it.
if kubectl -n cert-manager get deployment cert-manager \
    -o jsonpath='{.spec.template.spec.containers[0].args}' 2>/dev/null |
    grep -q -- '-certificaterequests-approver'; then
  ok "identity is attested (cert-manager's blanket approver is off)"
else
  bad "cert-manager's blanket approver is ON — every account can obtain any identity it asks for"
fi

if kubectl -n cert-manager get daemonset cert-manager-csi-driver-spiffe-driver \
    -o jsonpath='{.status.numberReady}' 2>/dev/null | grep -qE '^[1-9]'; then
  ok "identity driver"
else
  bad "identity driver — no pod can be given an identity"
fi

if [ "$fail" != 0 ]; then
  echo "the box is not usable: something above installed but is not there"
  exit 1
fi
echo "the box is usable"
