# Example with Ghost

[Ghost](https://ghost.org/) is a simple but powerfull blog engine. It is very nice to test some behaviors with Docker or Podman.

The given `docker-compose.yaml` file here declares a stand-alone blog service. To help using it, we use [Patwae](https://pathwae.net) reverse-proxy to listend http://ghost.example.localhost

The problem to solve is that the `url` environment variable correspond to the Ingress host when we will convert it to Helm Chart. So, we use the `mapenv` label to declare that `url` is actually `{{ .Values.blog.ingress.host }}` value.

Note that we also `ignore` pathwae because we don't need it in our Helm Chart.
