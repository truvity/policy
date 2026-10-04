# Workload prerequisites per mTLS level

**The rule.** A workload's mutual TLS has three levels, `off`, `identity` and
`enforced`, and each asks something different of the workload. This page says
what, level by level, and how to prove it. What the *platform* must provide
for each level is the other half, on the platform's side: see "truvity/openbao
docs: platform prerequisites" in the
[openbao repository's docs](https://github.com/truvity/openbao/tree/master/docs).
[transport-security.md](transport-security.md) says why the scheme is shaped
as it is; [platform.md §8](../contracts/platform.md) is the normative list of
what a platform must have.

**The levels, in chart terms.**

| Level | Chart `tls.mode` | What a pod serves | What a pod presents |
|---|---|---|---|
| off | `off` | cleartext | nothing |
| identity | `permissive` | cleartext and an authenticated port | an identity, wherever the callee authenticates |
| enforced | `strict` | the authenticated port only | an identity |

A level is **per serving component**, not per release. Each level contains
the one before it: do not skip one.

## Where to look

| Concern | Where |
|---|---|
| the `tls` block and its comments | the `tls:` block of [`values.yaml`](../../examples/url-shortener/charts/url-shortener/values.yaml) |
| one account per component | [`serviceaccount.yaml`](../../examples/url-shortener/charts/url-shortener/templates/serviceaccount.yaml) and [identity-and-secrets.md](identity-and-secrets.md) |
| the `tls` block of a component's ConfigMap | `_components.tpl` in the same chart |
| the permission to ask for an identity | [`identity-rbac.yaml`](../../examples/url-shortener/charts/url-shortener/templates/identity-rbac.yaml) |
| the proof on a cluster | [`identity-smoke.sh`](../../examples/url-shortener/hack/identity-smoke.sh) |
| the library, per language | the table in [transport-security.md](transport-security.md) |

## The chart's `tls` block

Four fields carry the whole decision.

| Field | Meaning |
|---|---|
| `tls.mode` | `off`, `permissive` or `strict`: the default for every serving component |
| `tls.components.<name>.mode` | the override for one serving component; unset means `tls.mode` |
| `tls.trustDomain` | the root of every identity the release admits; required once the mode is not `off`, because without it a peer from any trust domain would be admitted |
| `tls.peers.<name>` | the accounts that may call that component, from **outside** the release; the release's own callers are granted by the chart |

The rest (`csiDriver`, `mountPath`, `port`, `grantRequest`) only has to agree
with what the platform does. `tls.mode` stays `off` in the chart's defaults:
a chart is installable on a platform that provides none of this.

## Level off

**What it is.** Cleartext only, and no identity is requested. It is the
default and always will be.

**Prerequisites.**

- Nothing from the platform. No volume is mounted and no extra port is
  opened, so a pod can never wait for a volume nobody serves.
- Still one ServiceAccount per component (see below). It costs nothing here
  and is what the next level keys on; the render refuses two components
  sharing a name.

**Verify.**

1. Render the chart with defaults: no `tls` block appears in any
   component's ConfigMap, and no workload has the identity volume.
2. Install it on a cluster with none of the platform's identity machinery:
   every pod becomes ready.

## Level identity

**What it is.** `tls.mode: permissive`. A serving component serves both
cleartext and an authenticated port (a second port, hence two listeners),
and every workload that calls an authenticating callee presents its own
identity. It is a migration state and should carry a date: one edge
migrates at a time, in three commits and with no coordinated window (the
server adds its authenticated port, its clients move to it, the server
drops cleartext).

**Prerequisites.**

1. **One ServiceAccount per component, always.** An identity is namespace
   plus account:
   `spiffe://<trustDomain>/ns/<namespace>/sa/<account>`. Two components on
   one account are indistinguishable to an allow-list. The migration job
   gets its own account too. This is rule
   [C14](../contracts/component.md#c14-each-component-runs-as-its-own-serviceaccount).
2. **Permission for that account to ask for its identity.** The request is
   made *as* the account, which is the attestation. Without the permission
   the pod is never created, with the reason in an event nobody is watching.
   The chart grants it (`tls.grantRequest`); set it false where the platform
   grants it centrally.
3. **`tls.trustDomain`** set to the platform's trust domain.
4. **The platform provides the identity** (the list in
   [platform.md §8](../contracts/platform.md)): an attested identity, an
   approver that refuses a request for anyone else's, an authority, a trust
   bundle, and rotation. The workload cannot check any of that from
   inside; the verify step below is what proves it.

**What is injected per pod.**

| Injected | Detail |
|---|---|
| a CSI volume at `/var/run/identity` | `tls.crt`, `tls.key` and `ca.crt`, rotated about hourly. The pod's group must own the mount (set the pod's `fsGroup` to the user the image runs as) or the process cannot read its own certificate |
| the `tls` block of the app's ConfigMap | the three file paths, the trust domain, the allow-list, and for a serving component the authenticated address |
| one more port, in `permissive` only | the authenticated listener beside the cleartext one; under `strict` there is no second port |

Every workload mounts the volume once the transport is on at all, including
those with no mode of their own, because a caller must present an identity.

**What the workload's transport library must do.** The library is in the
table in [transport-security.md](transport-security.md); the duties are the
same in every language.

- **Hot reload.** Re-read the files when they change. A certificate read once
  authenticates fine until it rotates, then fails everywhere at once. A
  rotation caught half-done keeps serving the previous certificate.
- **Peer identity checks.** Verify the chain against `ca.crt`, then read the
  identity from the leaf and check it against the allow-list. The hostname is
  not the question. A leaf that is a certificate authority, or carries other
  than exactly one URI name, is refused whole. An empty allow-list admits
  nobody.
- **Client identity.** A caller presents its own certificate and verifies the
  callee's identity, by account, not by address.

**Verify.**

1. The pod has the volume and the three files exist, with a validity of
   about an hour (`openssl x509 -noout -dates -in /var/run/identity/tls.crt`
   from inside the pod).
2. The leaf's identity is the pod's own account, in the expected trust domain.
3. A call from an account that is on the allow-list succeeds on the
   authenticated port.
4. **The refusal, not the issuance.** An account asking for **another**
   account's identity must get nothing, and an account off the allow-list
   must be refused at the handshake. Issuance working proves nothing; this
   is the check [`identity-smoke.sh`](../../examples/url-shortener/hack/identity-smoke.sh)
   makes.
5. After an hour, calls still succeed with no restart.

## Level enforced

**What it is.** `tls.mode: strict` for a component: only the authenticated
port is served, and cleartext callers are refused.

**Prerequisites.**

1. **Everything above holds** for the component and for every caller.
2. **Callers must present identities before a callee goes strict.** Every
   caller is granted in the callee's allow-list (the release's own callers
   by the chart, anything from outside in `tls.peers.<name>`), already
   presents its identity, and already dials the authenticated port. Making
   the callee strict first turns every caller still on cleartext into an
   outage, and the error names a handshake, not a list nobody filled in.
   The e2e chart's Job and prober are callers too, and need granting
   before the flip, and their own `tls.peers` must name the accounts they
   call.
3. **A gateway-fronted component is never strict.** A gateway that
   terminates TLS at the edge and forwards cleartext has a strict callee
   refusing its only caller. The chart refuses it written down in
   `tls.components.<name>.mode` and refuses it inherited from a release-wide
   `strict`; set `tls.mode` to `permissive` and name the components that
   are strict instead.
4. **Probes stay reachable.** The probes listener is its own port and
   presents no identity; it is exempt, and no probe is allowed to demand
   one.

**Verify.**

1. A cleartext request to the component's old port is refused (connection
   refused or reset), and the authenticated port answers.
2. A caller with no identity is refused at the handshake, and the server's
   log, not the caller's error, says which rule refused it.
3. Every caller in the call graph still works: the release's smoke test
   passes after the flip.
4. A component fronted by a gateway still answers through the gateway, and
   renders `permissive`.

## Third-party servers

A product the platform runs but does not write (a database, a broker) takes
the same identity from the same volume, and these are the ways it may use it,
in order of preference.

1. **Native file reload.** The product re-reads the certificate files when
   they change. Point it at `/var/run/identity` and prefer this whenever the
   product can.
2. **A reload-helper sidecar**, only when the product cannot reload by
   itself: it watches the files and signals the product (or runs its reload
   command) when they change. It never sits in the traffic path and never
   terminates TLS.
3. **Never a proxy sidecar.** A terminating proxy in the pod moves identity
   out of the product, puts a second hop between the product and its callers
   and is a second thing to patch. Where a product cannot do any of the
   above, record a written exception naming the reason where the
   [conformance guide](conformance.md) says exceptions go.

**Identity-to-user mapping is the product's, and each product differs.**
Check the field it maps before granting anyone:

- **Postgres** maps the certificate's **common name** to a role, not the
  URI name carrying the account. An identity whose common name is empty or
  generic maps to no role or to the wrong one; give the product's mapping
  rule (an ident map, say) the field it actually reads.
- A product that reads the URI name or a hostname instead needs its own
  rule. Do not assume the library's account check applies: the product is
  doing the check, and you have to read which field.

A database whose operator already issues certificates its clients present
and maps them to roles by a different field has working mutual TLS of its
own; leave it.

**Verify (third party).** The files change on rotation and the product
serves the new certificate without a restart (check the served certificate's
serial before and after), and a client with the wrong identity is refused,
not mapped to a default role.

## Traps

**The platform half is invisible from here.** Certificates mount and the
logs say success while any account can still ask for any identity. Run the
refusal check.

**Setting `strict` release-wide.** It names the gateway-fronted component as
well, and the render refuses it. Name the components.

**Granting the callee before the caller can present.** The refusal appears
at the handshake with an error about a certificate. Look at the allow-list
first.

**Forgetting the group.** The permission error is on the CA file, and
nothing mentions identity.
