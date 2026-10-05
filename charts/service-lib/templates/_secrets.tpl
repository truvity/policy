{{/*
Secrets, as FILES (decision 0012, docs/contracts/config.md rule 5). A secret
is a NAME in the configuration (a field ending `Secret`), resolved by the
process through the one `secrets` source the file declares. This chart
delivers the `file` source: `platform.secretFiles` maps each name to the
Secret and key its value comes from, and the names appear as files under
`config.secrets.root`, mode 0440 (the pod's fsGroup reads them). A secret is
never an environment variable: there is no `valueFrom.secretKeyRef` here.

Refused, at render time and not in the pod:
  - `config.secrets.source: env`, which this chart cannot deliver;
  - `platform.secretFiles` without `config.secrets: {source: file, root: ...}`
    and an absolute root with no empty, `.` or `..` segment;
  - a declaration with no Secret or no key;
  - a name that is also the directory of another (`a` and `a/b`), or that is
    not a secret name (the pattern of schemas/fragments/secrets.json);
  - a `...Secret` field in `config` with no `config.secrets` block, which would
    start a pod that cannot read the secret it names;
  - `platform.volumes` with an entry named `secrets`, and a secrets root that
    equals, contains or is inside another mount the chart renders (the
    configuration directory, the identity, `platform.volumeMounts`).
*/}}
{{- define "service-lib.secretsSource" -}}
{{- ((.config | default dict).secrets | default dict).source | default "" -}}
{{- end -}}

{{/* "yes" when any key at any depth of the value ends in Secret. */}}
{{- define "service-lib.hasSecretField" -}}
{{- if kindIs "map" . -}}
{{- range $k, $v := . -}}
{{- if or (hasSuffix "Secret" $k) (include "service-lib.hasSecretField" $v) -}}yes{{- end -}}
{{- end -}}
{{- else if kindIs "slice" . -}}
{{- range $v := . -}}
{{- if include "service-lib.hasSecretField" $v -}}yes{{- end -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/* "yes" when the two absolute paths are equal or one contains the other. */}}
{{- define "service-lib.pathsOverlap" -}}
{{- $a := printf "%s/" (trimSuffix "/" .a) -}}
{{- $b := printf "%s/" (trimSuffix "/" .b) -}}
{{- if or (hasPrefix $a $b) (hasPrefix $b $a) -}}yes{{- end -}}
{{- end -}}

{{- define "service-lib.secretsRoot" -}}
{{- $p := .platform | default dict -}}
{{- $secrets := ($p.secretFiles | default dict) -}}
{{- $cfg := (.config | default dict).secrets | default dict -}}
{{- $source := $cfg.source | default "" -}}
{{- if eq $source "env" -}}
{{- fail (printf "component %s: config.secrets.source is env, which this chart does not deliver: a secret is a file (platform.secretFiles with config.secrets.source file)" .name) -}}
{{- end -}}
{{- if and (not $source) (include "service-lib.hasSecretField" (.config | default dict)) -}}
{{- fail (printf "component %s: config has a field ending Secret and no config.secrets block: declare config.secrets {source: file, root: ...} and the platform.secretFiles that fill it" .name) -}}
{{- end -}}
{{- range $v := ($p.volumes | default list) -}}
{{- if eq (get $v "name") "secrets" -}}
{{- fail (printf "component %s: platform.volumes has an entry named secrets, which is the library's own volume for platform.secretFiles" $.name) -}}
{{- end -}}
{{- end -}}
{{- if $secrets -}}
{{- if ne $source "file" -}}
{{- fail (printf "component %s: platform.secretFiles needs config.secrets.source: file, and config.secrets.source is %q" .name $source) -}}
{{- end -}}
{{- $root := $cfg.root | default "" -}}
{{- if not (regexMatch "^(/[A-Za-z0-9_][A-Za-z0-9_.-]*)+$" $root) -}}
{{- fail (printf "component %s: config.secrets.root must be an absolute directory with no empty, . or .. segment" .name) -}}
{{- end -}}
{{- $others := list (include "service-lib.configMountPath" .) -}}
{{- if include "service-lib.identityOn" . -}}
{{- $others = append $others ((($p.tls | default dict).mountPath) | default "/var/run/identity") -}}
{{- end -}}
{{- range $m := ($p.volumeMounts | default list) -}}
{{- $others = append $others (get $m "mountPath") -}}
{{- end -}}
{{- range $o := $others -}}
{{- if include "service-lib.pathsOverlap" (dict "a" $root "b" $o) -}}
{{- fail (printf "component %s: config.secrets.root %s overlaps the mount %s: a secrets directory shares no path with another mount" $.name $root $o) -}}
{{- end -}}
{{- end -}}
{{- $root -}}
{{- end -}}
{{- end -}}

{{- define "service-lib.secretsVolume" -}}
{{- $secrets := ((.platform | default dict).secretFiles | default dict) -}}
{{- if include "service-lib.secretsRoot" . }}
{{- $names := keys $secrets | sortAlpha -}}
{{- range $n := $names -}}
{{- $s := get $secrets $n -}}
{{- if not (regexMatch "^[A-Za-z0-9_][A-Za-z0-9_.-]*(/[A-Za-z0-9_][A-Za-z0-9_.-]*)*$" $n) }}
{{- fail (printf "component %s: platform.secretFiles.%q is not a secret name: a relative path of letters, digits, dots, underscores and dashes, which does not climb" $.name $n) }}
{{- end -}}
{{- if not (and $s.secretName $s.key) }}
{{- fail (printf "component %s: platform.secretFiles.%s needs both secretName and key" $.name $n) }}
{{- end -}}
{{- range $m := $names -}}
{{- if hasPrefix (printf "%s/" $n) $m }}
{{- fail (printf "component %s: platform.secretFiles.%s is a file and also the directory of %s" $.name $n $m) }}
{{- end -}}
{{- end -}}
{{- end }}
- name: secrets
  projected:
    defaultMode: 0440
    sources:
      {{- $done := dict }}
      {{- range $n := $names }}
      {{- $sn := (get $secrets $n).secretName }}
      {{- if not (hasKey $done $sn) }}
      {{- $_ := set $done $sn true }}
      - secret:
          name: {{ $sn | quote }}
          items:
            {{- range $m := $names }}
            {{- $t := get $secrets $m }}
            {{- if eq $t.secretName $sn }}
            - key: {{ $t.key | quote }}
              path: {{ $m | quote }}
            {{- end }}
            {{- end }}
      {{- end }}
      {{- end }}
{{- end }}
{{- end -}}

{{- define "service-lib.secretsMount" -}}
{{- with include "service-lib.secretsRoot" . }}
- name: secrets
  mountPath: {{ . }}
  readOnly: true
{{- end }}
{{- end -}}
