#!/usr/bin/env bash
# Asks each server in the box a REAL question, not "is the pod Running".
#
# This used to be two scripts — one that checked a CRD and a limit existed,
# a second that made an operator act on a resource and waited for it to
# become real. There is no operator here any more: every server the box
# carries is asked directly, so the two questions collapsed into one.
#
# A healthy pod answers none of these on its own. Measured here before this
# was written the other way: a JetStream server with no memory store
# configured reports healthy, accepts a connection, and refuses every memory
# stream with "insufficient memory resources available" — a chart that asks
# for one would fail in the box and work in production, which is the
# opposite of what a test environment is for.
set -euo pipefail
cd "$(dirname "$0")"
# shellcheck disable=SC1091
source versions.env

fail=0
ok()  { printf '  ok    %s\n' "$1"; }
bad() { printf '  FAIL  %s\n' "$1"; fail=1; }

step() { printf '\n\033[1m==> %s\033[0m\n' "$1"; }

step "the control plane: no leader election to lose"
# A single-node box has no second controller-manager or scheduler to hand
# off to, so leader election only adds a way to die: under host load the
# API server answers a lease renewal too slowly, the component exits on
# "leaderelection lost", and nothing reconciles until it restarts — no
# default ServiceAccount for a fresh namespace, no rollout progress.
# hack/kind/cluster.yaml turns it off; this asks the RUNNING pods, not the
# config that asked for it, on the same principle as every check above.
for component in kube-controller-manager kube-scheduler; do
  cmd=$(kubectl -n kube-system get pod -l "component=$component" \
    -o jsonpath='{.items[0].spec.containers[0].command}' 2>/dev/null || true)
  if echo "$cmd" | grep -q -- '--leader-elect=false'; then
    ok "$component runs with --leader-elect=false"
  else
    bad "$component is not running with --leader-elect=false: $cmd"
  fi
done

step "Postgres: a real query"
# The container is named explicitly: without it `kubectl exec` still works
# but prints "Defaulted container ..." on stdout ahead of the answer, which
# is indistinguishable from the server having said so.
if out=$(kubectl -n postgres exec deploy/postgres -c postgres -- \
    psql -U postgres -qtAX -c "select 6 * 7;" 2>&1) && [ "$(echo "$out" | tr -d '[:space:]')" = 42 ]; then
  ok "select 6 * 7 = 42"
else
  bad "the server did not answer a query it was asked: $out"
fi

step "NATS: a stream, a durable consumer, a publish, a consume"
# A throwaway client, the same way the S3 checks below exec into a pod
# rather than installing a client on the host. Everything it creates is
# named for THIS run and removed at the end, so the box is left as it was
# found.
#
# The consumer is created — durable, pull, "deliver all" — BEFORE the
# message is published, and the read is a bounded pull against that
# consumer, not a one-shot "get me the last message". `nats pub` is a plain
# core-NATS publish: it does not wait for JetStream to have stored the
# message, only for the server to have accepted it, so a publish is always
# followed by a short, asynchronous hop before the stream holds it. Reading
# immediately with no wait raced that hop; a durable consumer already
# waiting on the subject, fetched with an explicit timeout, does not — the
# fetch blocks until the message lands or the timeout is spent, instead of
# checking once and giving up.
ns=verify-$RANDOM
subject="verify.$ns"
if out=$(kubectl -n nats run "$ns" --rm -i --restart=Never --image "$NATS_BOX_IMAGE" --command -- sh -c "
    n() { nats --server nats://nats.nats.svc:4222 \"\$@\"; }
    n stream add '$ns' --subjects '$subject' --storage memory --retention limits --max-msgs=-1 --max-bytes=-1 --max-age=-1 --max-msg-size=-1 --discard=old --dupe-window=2m --defaults >/dev/null
    n consumer add '$ns' '$ns' --pull --deliver=all --ack=none --filter='$subject' --replicas=1 --defaults >/dev/null
    n pub '$subject' 'hello' >/dev/null
    got=0
    reply=\$(n consumer next '$ns' '$ns' --no-ack --timeout=10s 2>&1) || got=\$?
    echo \"\$reply\"
    if [ \$got -ne 0 ] || echo \"\$reply\" | grep -q 'Status: 408'; then
      echo '--- stream info, for a reader who was not there ---'
      n stream info '$ns' 2>&1 || true
      got=1
    fi
    n stream rm '$ns' -f >/dev/null 2>&1
    exit \$got
  " 2>&1) && echo "$out" | grep -q hello; then
  ok "a message published to $subject was read back from the stream"
else
  bad "the round trip through JetStream did not come back: $out"
fi

step "S3: a put and a get"
bucket="verify-$RANDOM"
if kubectl -n object-store exec deploy/s3 -- sh -c "
    set -e
    awslocal s3 mb s3://$bucket >/dev/null
    echo -n hello | awslocal s3 cp - s3://$bucket/hello.txt >/dev/null
    got=\$(awslocal s3 cp s3://$bucket/hello.txt - 2>/dev/null)
    awslocal s3 rb s3://$bucket --force >/dev/null
    [ \"\$got\" = hello ]
  " >/tmp/kind-verify-s3.log 2>&1; then
  ok "an object written to $bucket was read back"
else
  bad "the round trip through S3 did not come back: $(cat /tmp/kind-verify-s3.log)"
fi
rm -f /tmp/kind-verify-s3.log

step "the local registry: a push and a pull"
# A one-file image, built here rather than pulled, so this needs no network
# beyond the registry itself — the same reason the object store is pinned by
# digest rather than left to resolve.
img="localhost:${REGISTRY_PORT}/kind-verify:$RANDOM"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
echo ok >"$tmp/ok"
printf 'FROM scratch\nCOPY ok /ok\n' >"$tmp/Dockerfile"
if docker build -q -t "$img" "$tmp" >/dev/null 2>&1 &&
    docker push -q "$img" >/dev/null 2>&1; then
  node=$(kind get nodes --name "$CLUSTER" | head -1)
  if docker exec "$node" crictl pull "$img" >/dev/null 2>&1; then
    ok "$img was pushed from the host and pulled by a node"
  else
    bad "a node could not pull what was pushed to the registry"
  fi
  docker rmi "$img" >/dev/null 2>&1 || true
else
  bad "the host could not build or push to the local registry"
fi

if [ "$fail" != 0 ]; then
  echo
  echo "the box is not usable: something above did not answer"
  exit 1
fi
echo
echo "the box works: every server answered"
