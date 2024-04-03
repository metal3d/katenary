# Why those dependencies?

Katenary uses `compose-go` and several kubernetes official packages.

- `github.com/compose-spec/compose-go`: to parse compose files. It ensures that:
    - the project respects the "compose" specification
    - katenary uses the "compose" struct exactly the same way that podman-compose or docker does
- `github.com/spf13/cobra`: to parse command line arguments, subcommands and flags. It also generates completion for
  bash, zsh, fish and powershell.
- `github.com/thediveo/netdb`: to get the standard names of a service from its port number
- `gopkg.in/yaml.v3`:
    - to generate `Chart.yaml` and `values.yaml` files (only)
    - to parse Katenary labels in the compose file
- `k8s.io/api` and `k8s.io/apimachinery` to create Kubernetes objects
- `sigs.k8s.io/yaml`: to generate Katenary yaml files

