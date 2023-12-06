{{- define "cronjobs.fullname" -}}
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

{{- define "cronjobs.name" -}}
{{- if .Values.nameOverride -}}
{{- .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "cronjobs.labels" -}}
{{ include "cronjobs.selectorLabels" .}}
{{ if .Chart.Version -}}
{{ printf "katenary.v3/chart-version: %s" .Chart.Version }}
{{- end }}
{{ if .Chart.AppVersion -}}
{{ printf "katenary.v3/app-version: %s" .Chart.AppVersion }}
{{- end }}
{{- end -}}

{{- define "cronjobs.selectorLabels" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{ printf "katenary.v3/name: %s" $name }}
{{ printf "katenary.v3/instance: %s" .Release.Name }}
{{- end -}}
