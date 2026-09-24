# RPC and schemas

**The rule.** A boundary of ownership is an RPC. One service owns a table, a
lifecycle or a decision, and everything else asks it. The schema lives in the
repository with a build configuration beside it, both sides are generated
from it, and the generated code is committed.

**Why.** Two components writing the same table are two components that have
to agree about it forever, in code that never mentions the other. The
agreement is not written down anywhere, so it is discovered when one of them
changes — by the other one breaking, at run time, in a way that reads as data
corruption rather than as a contract violation. Putting a boundary in front
of the table turns that into a compile error on one side and a version
negotiation on the other.

It also moves the credential. In this example the counter used to hold a
database password with write rights on a table it did not own; now it holds
an address. A component that cannot write the table cannot write it wrongly,
and the rights it was granted stop being a thing anyone has to reason about.

## Which shape, and when

| Shape | When | In the example |
|---|---|---|
| RPC | a boundary of ownership — one service owns it, the rest ask | `stat` calls `UrlsService.RecordClick` |
| Event | fan-out: several unrelated things must happen after a fact | `redirect` publishes, `stat` and `log` consume |
| Direct read | only where latency justifies a second reader, and written down | `redirect` reads the table on the hot path |

The third row is the one that needs saying out loud, because "everything goes
through the boundary" is a rule people adopt and then quietly break. The
redirect path is the example's hot path and a second network hop on it is not
worth what it buys, so it reads the table directly — and that is recorded
here and in the example's README rather than left for a reader to find and
assume is a mistake.

## Where to look

| Language | Server | Client |
|---|---|---|
| Go | [`cmd/urls/main.go`](../../examples/url-shortener/cmd/urls/main.go), one handler in [`internal/business/urls/`](../../examples/url-shortener/internal/business/urls/) | [`internal/business/stat/remote.go`](../../examples/url-shortener/internal/business/stat/remote.go) |
| TypeScript | — | arrives with the front end |
| Python | not yet; the ecosystem's support is younger. See [canon/python.md](../canon/python.md) | |
| Kotlin | client-only by design | arrives with the counter's rewrite |

The schema is [`proto/urlshortener/v1/`](../../examples/url-shortener/proto/urlshortener/v1/),
the generation is configured in `buf.gen.yaml`, and the output is committed
under `internal/gen/`.

## One handler, three protocols

The server is one Connect handler and it serves gRPC, gRPC-Web and Connect on
the same port. The protocol is the caller's choice, not the service's: an
in-cluster client speaks gRPC, a browser speaks gRPC-Web or Connect, and a
person debugging speaks Connect with `curl`, because a unary Connect call is
an ordinary POST with a JSON body.

That last one matters more often than it sounds. It is the difference between
a boundary you can poke at three in the morning and one you need a generated
client to ask a question of.

## Traps

**A gRPC client in the cluster needs HTTP/2, and with the transport off there
is no TLS to negotiate it over.** Both ends have to be told to speak HTTP/2
in cleartext. If only the client is, the server answers HTTP/1.1 and every
gRPC call fails at the handshake — while Connect calls keep working, which
makes it look like the client's fault. Since Go 1.24 this is
`http.Server.Protocols` and `http.Transport.Protocols`; the `h2c` wrapper
that used to be the way is deprecated in favour of them.

**Generated code that is not committed breaks the editor first.** The
compiler can be given a build step; an editor resolving a symbol cannot. Commit
it and fail a gate on a diff against a regeneration.

**A field renumbered by hand is a wire incompatibility no compiler catches**,
because both sides are regenerated from the same file in the same commit and
agree with each other perfectly. That is what the schema linter and the
breaking-change check are for, and it is why the schema is a file in the
repository rather than a description in a document.

**An interface declared next to its implementation carries the whole
implementation's surface.** The counter's `ClickCounter` is declared at the
consumer and has one method, which is why swapping a database write for an
RPC was a new file and a changed line rather than a rewrite. Declare what you
need where you need it.

**Only "not found" is worth distinguishing.** Every other storage failure is
the service's own problem, and a caller that could tell a constraint
violation from a dropped connection would start depending on which it got.
