Thanks to have installed {{ .Chart.Name }} {{ .Chart.Version }} as {{ .Release.Name }} ({{.Chart.AppVersion }}).

# Get release information

To learn more about the release, try:

  $ helm -n {{ .Release.Namespace }} status {{ .Release.Name }}
  $ helm -n {{ .Release.Namespace }} get values {{ .Release.Name }}
  $ helm -n {{ .Release.Namespace }} get all {{ .Release.Name }}

# To delete the release

Use helm uninstall command to delete the release. 

  $ helm -n {{ .Release.Namespace }} uninstall {{ .Release.Name }}

Note that some resources may still be in use after a release is deleted. For exemple, PersistentVolumeClaims are not deleted by default for some storage classes or if some annotations are set.

# More information

You can see this notes again by running:

  $ helm -n {{ .Release.Namespace }} get notes {{ .Release.Name }}

{{- $count := 0 -}}
{{- $listOfURL := "" -}}
{{* DO NOT REMOVE, replaced by notes.go: ingress_list *}}
{{- if gt $count 0 }}

# List of activated ingresses URL:
{{ $listOfURL }}

You can get these urls with kubectl:

    kubeclt get ingress -n {{ .Release.Namespace }}

{{- end }}

Thanks for using Helm!
