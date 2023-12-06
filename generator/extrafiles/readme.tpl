# {{ .Chart.Name }}

{{ .Chart.Description }}

## Installing the Chart

To install the chart with the release name `my-release`:

```bash
# Standard Helm install
$ helm install  my-release {{ .Chart.Name }}

# To use a custom namespace and force the creation of the namespace
$ helm install my-release --namespace my-namespace --create-namespace {{ .Chart.Name }}

# To use a custom values file
$ helm install my-release -f my-values.yaml {{ .Chart.Name }}
```

See the [Helm documentation](https://helm.sh/docs/intro/using_helm/) for more information on installing and managing the chart.

## Configuration

The following table lists the configurable parameters of the {{ .Chart.Name }} chart and their default values.

| {{ printf "%-*s" .DescrptionPadding "Parameter" }} | {{ printf "%-*s" .DefaultPadding "Default" }} |
| {{ repeat "-" .DescrptionPadding  }} | {{ repeat "-" .DefaultPadding }} |
{{- range .Chart.Values }}
{{ . }}
{{- end }}


