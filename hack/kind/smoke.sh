#!/usr/bin/env bash
# Proves the box does the thing a container cannot: an operator ACTS.
#
# verify.sh asks whether each CRD is installed, which is necessary and says
# nothing about whether anything is watching. A stream resource applied to a
# cluster with no controller is accepted, stored, and never becomes a stream;
# a database resource with no operator is a row in etcd. Both look exactly
# like success from the chart's side, which is why the chart tests run here.
#
# Everything is created in its own namespace and removed at the end, so the
# box is left as it was found.
set -euo pipefail

ns=smoke-$RANDOM
cleanup() { kubectl delete namespace "$ns" --wait=false >/dev/null 2>&1 || true; }
trap cleanup EXIT

kubectl create namespace "$ns" >/dev/null
echo "==> namespace $ns"

echo "==> a database becomes a database"
kubectl -n "$ns" apply -f - >/dev/null <<YAML
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: smoke
spec:
  instances: 1
  storage:
    size: 256Mi
YAML
kubectl -n "$ns" wait cluster/smoke --for=condition=Ready --timeout=6m
kubectl -n "$ns" get secret smoke-app -o jsonpath='{.data.uri}' >/dev/null
echo "    the operator created the cluster and its credentials"

echo "==> a stream becomes a stream"
kubectl -n "$ns" apply -f - >/dev/null <<YAML
apiVersion: jetstream.nats.io/v1beta2
kind: Stream
metadata:
  name: smoke
spec:
  name: SMOKE
  subjects: ["smoke.>"]
  storage: memory
  servers: ["nats://nats.nats.svc:4222"]
YAML
# The controller reports Ready only once the server has the stream, which is
# the assertion worth making: the resource existing proves nothing.
kubectl -n "$ns" wait stream/smoke --for=condition=Ready --timeout=2m
echo "    the controller created the stream on the server"

echo "==> a bucket becomes a bucket"
kubectl -n object-store exec deploy/s3 -- awslocal s3 mb "s3://smoke-$ns" >/dev/null
kubectl -n object-store exec deploy/s3 -- awslocal s3 ls | grep -q "smoke-$ns"
kubectl -n object-store exec deploy/s3 -- awslocal s3 rb "s3://smoke-$ns" >/dev/null
echo "    the object store accepted a bucket"

echo "the box works: every operator acted"
