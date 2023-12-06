# somevolumes

A Helm chart for somevolumes

## Installing the Chart

To install the chart with the release name `my-release`:

```bash
# Standard Helm install
$ helm install  my-release somevolumes

# To use a custom namespace and force the creation of the namespace
$ helm install my-release --namespace my-namespace --create-namespace somevolumes

# To use a custom values file
$ helm install my-release -f my-values.yaml somevolumes
```

See the [Helm documentation](https://helm.sh/docs/intro/using_helm/) for more information on installing and managing the chart.

## Configuration

The following table lists the configurable parameters of the somevolumes chart and their default values.

| Parameter                                       | Default           |
| ----------------------------------------------- | ----------------- |
| `site1.imagePullPolicy`                         | `IfNotPresent`    |
| `site1.persistence.statics.accessMode[0].value` | `ReadWriteOnce`   |
| `site1.persistence.statics.enabled`             | `true`            |
| `site1.persistence.statics.size`                | `1Gi`             |
| `site1.persistence.statics.storageClass`        | `-`               |
| `site1.replicas`                                | `1`               |
| `site1.repository.image`                        | `docker.io/nginx` |
| `site1.repository.tag`                          | `1`               |


