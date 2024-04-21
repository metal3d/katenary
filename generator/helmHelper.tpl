{{- define "__APP__.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "__APP__.name" -}}
{{- if .Values.nameOverride -}}
{{- .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "__APP__.labels" -}}
{{ include "__APP__.selectorLabels" .}}
{{ if .Chart.Version -}}
{{ printf "__PREFIX__chart-version: '%s'" .Chart.Version }}
{{- end }}
{{ if .Chart.AppVersion -}}
{{ printf "__PREFIX__app-version: '%s'" .Chart.AppVersion }}
{{- end }}
{{- end -}}

{{- define "__APP__.selectorLabels" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{ printf "__PREFIX__name: %s" $name }}
{{ printf "__PREFIX__instance: %s" .Release.Name }}
{{- end -}}
