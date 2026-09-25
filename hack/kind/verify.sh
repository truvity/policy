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

step "NATS: a stream, a publish, a consume"
# A throwaway client, the same way the S3 checks below exec into a pod
# rather than installing a client on the host. Everything it creates is
# named for THIS run and removed at the end, so the box is left as it was
# found.
ns=verify-$RANDOM
subject="verify.$ns"
if out=$(kubectl -n nats run "$ns" --rm -i --restart=Never --image "$NATS_BOX_IMAGE" --command -- sh -c "
    n() { nats --server nats://nats.nats.svc:4222 \"\$@\"; }
    n stream add '$ns' --subjects '$subject' --storage memory --retention limits --max-msgs=-1 --max-bytes=-1 --max-age=-1 --max-msg-size=-1 --discard=old --dupe-window=2m --defaults >/dev/null
    n pub '$subject' 'hello' >/dev/null
    n stream get '$ns' --last-for '$subject' 2>/dev/null
    got=\$?
    n stream rm '$ns' -f >/dev/null
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
