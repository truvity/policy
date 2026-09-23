# Kotlin and JVM canon

**Status: stub.** This page is written from the first JVM service held to
these contracts, not before it.

That order is deliberate. A canon assembled from a survey of what is popular
describes nobody's service; one assembled from a service that had to satisfy
the contracts describes the choices that actually came up. The Go canon was
written the second way and is short because of it.

## What is already fixed

The [service contract](../contracts/service.md) is language-neutral and binds
a JVM service exactly as it binds any other: one configuration file validated
against a schema, secrets from the environment, probes on their own listener
at `/health/live` and `/health/ready`, JSON logs on stdout at one level, a
drain on `SIGTERM`, a version read from the build, and a runtime image with
no build step in it — a jar copied onto a JRE base, every platform in one
job.

[0001](../decisions/0001-no-di-containers.md) applies at the same boundary it
applies everywhere: a framework may wire itself however it likes, and the
service's own dependencies are constructed where they can be read.

## What this page will name

Build tool, HTTP server, JSON, logging, test libraries, and the
configuration loader — each seeded from what the adopting service already
uses, unless it conflicts with a contract above.

## One thing worth knowing in advance

The Connect ecosystem's Kotlin library is a **client** library: it generates
clients for Connect, gRPC and gRPC-Web, and no servers. A JVM service that
consumes an RPC boundary is well served by it. A JVM service that *owns* one
serves gRPC with a JVM gRPC stack, which Connect clients in every other
language reach over their gRPC transport, and which a browser reaches through
a gateway's gRPC-Web filter.

That is a real constraint on where a JVM service sits in a topology, and it
is better known before the first one is written than after.
