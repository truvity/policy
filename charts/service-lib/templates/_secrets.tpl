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
  - a name that is also the directory of another (`a` and `a/b`).
*/}}
{{- define "service-lib.secretsSource" -}}
{{- ((.config | default dict).secrets | default dict).source | default "" -}}
{{- end -}}

{{- define "service-lib.secretsRoot" -}}
{{- $secrets := ((.platform | default dict).secretFiles | default dict) -}}
{{- $cfg := (.config | default dict).secrets | default dict -}}
{{- if eq ($cfg.source | default "") "env" -}}
{{- fail (printf "component %s: config.secrets.source is env, which this chart does not deliver: a secret is a file (platform.secretFiles with config.secrets.source file)" .name) -}}
{{- end -}}
{{- if $secrets -}}
{{- if ne ($cfg.source | default "") "file" -}}
{{- fail (printf "component %s: platform.secretFiles needs config.secrets.source: file, and config.secrets.source is %q" .name ($cfg.source | default "")) -}}
{{- end -}}
{{- $root := $cfg.root | default "" -}}
{{- if not (regexMatch "^(/[A-Za-z0-9_][A-Za-z0-9_.-]*)+$" $root) -}}
{{- fail (printf "component %s: config.secrets.root must be an absolute directory with no empty, . or .. segment" .name) -}}
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
          name: {{ $sn }}
          items:
            {{- range $m := $names }}
            {{- $t := get $secrets $m }}
            {{- if eq $t.secretName $sn }}
            - key: {{ $t.key }}
              path: {{ $m }}
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
