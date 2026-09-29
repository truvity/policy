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
| Go | [`internal/events/publisher.go`](../../examples/url-shortener/internal/events/publisher.go), called from the redirect service | |
| TypeScript | | |
| Kotlin | | the counter's durable consumer, in [`Stat.kt`](../../examples/url-shortener/stat/src/main/kotlin/com/truvity/example/stat/Stat.kt) |
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

### Authenticating with the workload's own certificate instead

A publisher can present its **workload identity** to the broker instead of a
token: `events.nats.tls` in its configuration (`events.tls` in the chart). The
client dials TLS, presents the certificate the platform mounted for the
service's own `tls` block, and verifies the broker's certificate against a
trust bundle of its own. The broker maps the identity in the certificate to a
user with its own permissions, so what the service may publish is the
broker's decision and nothing in the service's file.

- **The certificate is the only credential.** The chart sends no token
  beside it. A certificate the broker cannot map must fail at connect; if
  the token followed as a fallback, an unmappable certificate would succeed
  as the token's account with that account's full rights, and the setup
  would read as working.
- **It needs an identity.** `tls.mode` for the component must not be `off`;
  the helper refuses to build without one, in words that name the setting,
  rather than dialling TLS with no client certificate.
- **Rotation drops nothing.** The certificate is re-read on every connect
  (`transport.Identity.ClientTo`, which loads it the way the listener side
  does), and the broker checks it at the handshake only, so a connection
  already made survives the platform's rotation and the next connect
  presents the new one. `transport/client_to_test.go` holds both halves.
- **The broker's name is not the workload identities'.** A broker has a name
  and no workload identity, and usually a certificate from a different chain,
  so the trust bundle (`caFile`) and the name to verify (`serverName`, when
  it is not the host in `url`) are its own settings.
- **Only a service that declares it acts on it.** The fragment admits the key
  in every schema that references it; today only the URL shortener's
  redirect reads it.

## What travels

The body is the fact. Its **type and origin travel as message headers**, not
inside the body, so a consumer can route on them without parsing a payload
it may not understand.

**The trace context travels in the message too**, as a lower-case
`traceparent` header, so a consumer's span can be a child of the request
that caused the event. A batching consumer links instead of parenting. See
[logging-and-telemetry.md](logging-and-telemetry.md#a-trace-that-stays-whole).

## Naming

**A stream's name, its subjects and its durable consumer names are
cluster-global, not namespace-scoped.** They live on the broker, and the
broker has never heard of a Kubernetes namespace — nothing about how
Kubernetes scopes its own objects protects one install's stream from
another's.

The rule: every one of those names derives from **this install's namespace
and its install name, together**. Namespace alone collides every install
sharing a namespace — two CI runs against one namespace, each installing
under the application's own default release name. Install name alone
collides two installs that happen to agree on a name in different
namespaces — two engineers who each call their own copy by the project's
name. The pair is the smallest thing that separates both.

The example's two charts compute these names from one shared formula
(`installName`, defaulting to the release name, in
[`_helpers.tpl`](../../examples/url-shortener/charts/url-shortener-infra/templates/_helpers.tpl))
instead of taking them as values — see
[`platform.md` rule 6](../contracts/platform.md#6-streams-and-databases-are-things-the-service-finds-not-things-it-makes)
for why a value is the wrong fix here: the name is this project's own
convention, not a platform's.

## Traps

**A service does not create the stream it reads.** A consumer that creates a
missing stream will, one day, create it with the wrong retention on a cluster
where it was deliberately absent, and nothing will report it.

**A stream named by namespace only, or by install name only, still
collides.** Both installs render, both install, and one consumer quietly
reads the other's events — or replays what it already saw, or both — and
nothing reports it, because every health check both installs run stays
green. The chart test that catches it renders two installs varied one way
and then the other and asserts neither shares a name with the other.

**Never give up reconnecting.** A broker restart is an ordinary event; a
service that exits on one turns a blip into a rollout.

**Drain, do not close.** On shutdown, a drain finishes what is in flight. A
close cuts it, and the messages come back later as duplicates — which is the
consumer's problem to handle anyway, but not one to cause on purpose.

**Connecting with no credential works until it does not.** A broker with
authorisation not yet enforced accepts anonymous clients, so the omission is
invisible until the day it is turned on, and then every such service fails at
once.

**`nats-py`'s own reconnect logic never runs on an expired credential.** The
Go and Kotlin clients on this same broker treat the server closing a
connection over a rotated credential as a retryable error and reconnect on
their own; `nats-py` 2.x closes the client outright on that one server error
instead — `max_reconnect_attempts` and `allow_reconnect` are never
consulted — so every fetch after that raises `ConnectionClosedError`
forever. The Python consumer dials again itself when that one error
appears; see
[`pull.pull_or_reconnect`](../../examples/url-shortener/log/src/url_shortener_log/pull.py).
