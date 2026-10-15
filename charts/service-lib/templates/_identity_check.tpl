{{/*
The identity check: a Job that proves a pod admitted NOW gets the component's
cloud identity, before the release's workloads are created.

WHY IT EXISTS. On EKS a Pod Identity association is "Synced" (the AWS API
returned) seconds before the pod-admission webhook can see it; AWS documents
the association as eventually consistent. Credentials are injected ONLY when a
pod is admitted, so a pod created in that gap runs without them for its whole
life, and restarting its containers never admits it again. Waves, hook order
and health checks all wait on the resource's status, which is already "Synced".
This Job is the one thing that waits on what matters: it asks STS who it is,
and `restartPolicy: Never` makes every retry a NEW pod, admitted afresh. The
release's workloads are created only after a pod has been admitted with the
identity. docs/decisions/0013-pods-are-admitted-with-their-identity.md.

It is a Helm hook, `pre-install,pre-upgrade`, which Argo CD maps to PreSync and
a Helm-based controller runs as Helm does. A hook runs before anything else in
the release, so the ServiceAccount must already exist: it is the
infrastructure chart's (platform contract, rule 11), never this release's.

Takes the component spec, plus:

    cloudIdentity  true when the component's account carries a cloud identity;
                   false or absent renders NOTHING. The including chart derives
                   it from the tier, so there is no switch for a deployer.
    expectRole     optional role NAME the pod must have assumed.
    image          optional {registry, repository, tag, digest}; absent, the
                   chart's own top-level `images.identity-check`.

The pod carries the component's own labels, deliberately: a network policy
that lets the component reach the cloud API lets this check reach it too, and
a check the policy blocks would fail for a reason the component never has.
*/}}
{{- define "service-lib.identityCheck" -}}
{{- if .cloudIdentity -}}
{{- $p := .platform | default dict -}}
{{- $image := dict "context" .context "name" "identity-check" "platform" (dict "image" (.image | default dict)) -}}
apiVersion: batch/v1
kind: Job
metadata:
  name: {{ include "service-lib.fullname" . }}-identity-check
  labels: {{- include "service-lib.componentLabels" . | nindent 4 }}
  annotations:
    "helm.sh/hook": pre-install,pre-upgrade
    "helm.sh/hook-weight": "-20"
    "helm.sh/hook-delete-policy": before-hook-creation
spec:
  backoffLimit: 6
  activeDeadlineSeconds: 300
  template:
    metadata:
      labels: {{- include "service-lib.componentLabels" . | nindent 8 }}
    spec:
      restartPolicy: Never
      serviceAccountName: {{ include "service-lib.serviceAccountName" . }}
      {{- include "service-lib.podSecurityContext" . | nindent 6 }}
      containers:
        - name: identity-check
          image: {{ include "service-lib.image" $image }}
          imagePullPolicy: {{ $p.imagePullPolicy | default "IfNotPresent" }}
          {{- include "service-lib.containerSecurityContext" . | nindent 10 }}
          {{- with .expectRole }}
          args: ["-expect-role", {{ . | quote }}]
          {{- end }}
          resources:
            requests:
              cpu: 10m
              memory: 16Mi
            limits:
              memory: 64Mi
{{- end -}}
{{- end -}}
