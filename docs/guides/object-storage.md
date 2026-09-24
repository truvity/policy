# Object storage

**The rule.** A store is an endpoint, not a vendor. A service is given a
bucket name, a region, an endpoint, a path-style flag and — where the
platform cannot supply ambient credentials — the NAMES of two environment
variables. Nothing in the service knows which implementation answered.

**Why.** A service that hard-codes a vendor cannot be tested without that
vendor, and "cannot be tested without it" quietly becomes "cannot be run
anywhere else". The five keys above reach a cloud service, a store inside
somebody's own cluster, and a test double in a local box, and the component
cannot tell them apart. The example proves that by pointing the archiver at
the local cluster's own store and changing nothing in the code.

It also keeps the credential out of everything. The configuration file
carries the names of the variables; the values arrive in the environment from
a Secret. A file is mounted from a config map, printed when somebody debugs a
deployment, and committed as a fixture — it has to survive all three being
true.

## The shape

The `bucket` fragment, in [`schemas/fragments/bucket.json`](../../schemas/fragments/bucket.json):

| Key | For |
|---|---|
| `name` | the bucket, which **exists already** — a service does not create its own store |
| `region` | where it is |
| `endpoint` | unset means the SDK's own resolution; set means anything else |
| `ca` | a bundle the platform mounts, for a store whose endpoint is not signed by a public root — the ordinary case inside somebody's network, not the exotic one |
| `pathStyle` | required by most non-cloud implementations |
| `credentialsEnv` | the NAMES of two variables. **Unset is the better answer**: it means the SDK's ambient credentials, which is what a workload identity provides and what leaves nothing to leak |

## Where to look

| Language | Client | Configuration |
|---|---|---|
| Python | [`archive.py`](../../examples/url-shortener/log/src/url_shortener_log/archive.py) — one `put_object`, behind a one-method protocol | [`config.py`](../../examples/url-shortener/log/src/url_shortener_log/config.py) |
| Go | follows, with the first Go component that stores something | |
| Kotlin | the cloud vendor's v2 SDK; see [canon/kotlin.md](../canon/kotlin.md) | |
| TypeScript | follows | |

The chart renders it in [`config.yaml`](../../examples/url-shortener/charts/url-shortener/templates/config.yaml),
and the local cluster's store is an ordinary Deployment in
[`hack/kind/localstack.yaml`](../../hack/kind/localstack.yaml).

## A one-method interface is the point

The archiver's client is declared at the consumer with a single method:

```python
class ObjectStore(Protocol):
    def put_object(self, *, Bucket: str, Key: str, Body: bytes, ContentType: str) -> object: ...
```

The vendor's own client satisfies it without knowing about it, and a test
substitutes five lines instead of a mock. It also stops a caller reaching for
a delete, a lifecycle rule or a bucket creation — operations that belong to
whoever provisioned the store.

## Traps

**A service that can create a bucket can create it in the wrong account**,
with the wrong retention, and nothing notices until somebody looks. The
bucket exists already; the service is told its name.

**A bucket is usually shared, so write under a prefix.** A component that
writes to the root of one cannot be granted permission to write only its own
objects.

**Name an object by something that does not vary on a retry.** The archiver
keys each object on the first stream sequence in the batch and nothing else:
a batch is acknowledged only after its object is written, so a failed write
means the same records are redelivered — and a redelivery begins at the same
sequence and overwrites its own partial attempt instead of leaving a second
copy beside it. Putting the count or the wall clock in the key would lose
that.

**Acknowledge after storing, never before.** A consumer that acknowledged
what it had not stored has lost it for good, and the broker will not send it
again. The archiver's failed write KEEPS the batch, and there is a test for
exactly that.

**A store that is down is not your component's outage to report.** The
archiver reaches the bucket once at start-up, so a bucket nobody granted
access to is a refusal then rather than a surprise an hour later when the
first batch fills — but readiness afterwards reports whether it can consume,
not whether the store is up.
