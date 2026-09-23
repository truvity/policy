# Events

**The rule.** Fan-out is an event: the publisher states a fact and the
consumers are none of its business. A service connects to a stream it did not
create. [service.md §8](../contracts/service.md) and
[platform.md §6](../contracts/platform.md).

**Why.** When several unrelated things must happen after a fact, an RPC per
consumer makes the publisher own a list it cannot maintain. An event makes
the fact the interface. The stream itself belongs to the infrastructure
release, because it outlives any one version of the application and its
retention is an operational decision.

## Where to look

| Language | Publish | Consume |
|---|---|---|
| Go | [`internal/events/publisher.go`](../../examples/url-shortener/internal/events/publisher.go), called from the redirect service | the durable consumer in the counter's `main` |
| TypeScript | follows | follows |
| Kotlin | | the counter, once rewritten |
| Python | | the pull consumer in [`__main__.py`](../../examples/url-shortener/log/src/url_shortener_log/__main__.py): durable by name, acknowledged only after the batch is stored |

The connection helper, including authentication, is
[`internal/runtime/natstoken.go`](../../examples/url-shortener/internal/runtime/natstoken.go).
The stream is declared by the infrastructure chart, not the application one.

## Authentication

The client reads a **token from a file the platform mounts**, and re-reads it
on every reconnect. The token is the workload's own account token; the broker
hands it to an authorisation service, which answers from the account the pod
actually runs as rather than from anything the client claims.

**Re-read, not read once.** This is the whole reason the helper installs a
handler rather than passing a string. A short-lived token read at start-up
authenticates fine until the first reconnect, and then fails somewhere far
from the line that read it.

## What travels

The body is the fact. Its **type and origin travel as message headers**, not
inside the body, so a consumer can route on them without parsing a payload
it may not understand.

## Traps

**A service does not create the stream it reads.** A consumer that creates a
missing stream will, one day, create it with the wrong retention on a cluster
where it was deliberately absent, and nothing will report it.

**Never give up reconnecting.** A broker restart is an ordinary event; a
service that exits on one turns a blip into a rollout.

**Drain, do not close.** On shutdown, a drain finishes what is in flight. A
close cuts it, and the messages come back later as duplicates — which is the
consumer's problem to handle anyway, but not one to cause on purpose.

**Connecting with no credential works until it does not.** A broker with
authorisation not yet enforced accepts anonymous clients, so the omission is
invisible until the day it is turned on, and then every such service fails at
once.
