Your release is named {{ .Release.Name }}.

To learn more about the release, try:

  $ helm -n {{ .Release.Namespace }} status {{ .Release.Name }}
  $ helm -n {{ .Release.Namespace }} get all {{ .Release.Name }}

To delete the release, run:

  $ helm -n {{ .Release.Namespace }} delete {{ .Release.Name }}

You can see this notes again by running:

  $ helm -n {{ .Release.Namespace }} get notes {{ .Release.Name }}

{{- $count := 0 -}}
{{- range $s, $v := .Values -}}
{{- if and $v $v.ingress -}}
{{- $count = add $count 1 -}}
{{- if eq $count 1 }}

The ingress list is:
{{ end }}
  - {{ $s }}: http://{{ $v.ingress.host }}{{ $v.ingress.path }}
{{- end -}}
{{ end -}}

