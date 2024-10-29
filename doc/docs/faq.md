# Frequently Asked Questions

## Why Katenary?

The main author[^1] of Katenary is a big fan of Podman, Docker and makes a huge use of Compose. He uses it a lot in his
daily work. When he started to work with Kubernetes, he wanted to have the same experience as with Docker Compose.
He wanted to have a tool that could convert his `docker-compose` files to Kubernetes manifests, but also to Helm charts.

Kompose was a good option. But the lacks of some options and configuration for the output Helm chart made him think
about creating a new tool. He wanted to have a tool that could generate a complete Helm chart, with a lot of options
and flexibility.

[^1]: I'm talking about myself :sunglasses: - Patrice FERLET, aka Metal3d, Tech Lead and DevOps Engineer at Klee Group.

## What's the difference between Katenary and Kompose?

[Kompose](https://kompose.io/) is a very nice tool, made by the Kubernetes community. It's a tool to convert
`docker-compose` files to Kubernetes manifests. It's a very good tool, and it's more mature than Katenary.

Kompose is able to generate Helm charts, but [it could be not the case in future releases](https://github.com/kubernetes/kompose/issues/1716) for several reasons[^2].

[^2]: The author of Kompose explains that they have no bandwidth to maintain the Helm chart generation. It's a complex
task, and we can confirm. Katenary takes a lot of time to be developed and maintained. This issue mentions Katenary as
an alternative to Helm chart generation :smile:

The project is focused on Kubernetes manifests and proposes to use "kusomize" to adapt the manifests. Helm seems to be
not the priority.

Anyway, before this decision, the Helm chart generation was not what we expected. We wanted to have a more complete
chart, with more options and more flexibility.

> That's why we decided to create Katenary.

Kompose didn't manage to generate a values file, complex volume binding, and many other things. It was also not able
to manage dependencies between services.

> Be sure that we don't want to compete with Kompose. We just want to propose a different approach to the problem.

Kompose is an excellent tool, and we use it in some projects. It's a good choice if you want to convert
your `docker-compose` files to Kubernetes manifests, but if you want to use Helm, Katenary is the tool you need.

## Why not using "one label" for all the configuration?

That was a dicsussion I had with my colleagues. The idea was to use a single label to store all the configuration.
But, it's not a good idea.

Sometimes, you will have a long list of things to configure, like ports, ingress, dependencies, etc. It's better to have
a clear and readable configuration. Segmented labels are easier to read and to maintain. It also avoids having too
many indentation levels in the YAML file.

It is also more flexible. You can add or remove labels without changing the others.

## Why not using a configuration file?

The idea was to keep the configuration at a same place, and using the go-compose library to read the labels. It's
easier to have a single file to manage.

By the way, Katenary auto accepts a `compose.katenary.yaml` file in the same directory. It's a way to separate the
configuration from the compose file. It uses
the [overrides' mechanism](https://docs.docker.com/compose/multiple-compose-files/merge/) like "compose" does.

## Why not developing with Rust?

Seriously...

OK, I will answer.

Rust is a good language. But, Podman, Docker, Kubernetes, Helm, and mostly all technologies around Kubernetes are
written in Go. We have a large ecosystem in Go to manipulate, read, and write Kubernetes manifests as parsing
Compose files.

> Go is better for this task.

There is no reason to use Rust for this project.

## Any chance to have a GUI?

Yes, it's a possibility. But, it's not a priority. We have a lot of things to do before. We need to stabilize the
project, to have a good documentation, to have a good test coverage, and to have a good community.

But, in a not so far future, we could have a GUI. The choice of [Fyne.io](https://fyne.io) is already made and we tested some concepts.

## I'm rich (or not), I want to help you. How can I do?

You can help us in many ways.

- The first things we really need, more than money, more than anything else, is to have feedback. If you use Katenary,
if you have some issues, if you have some ideas, please open an issue on the [GitHub repository](https://github.com/metal3d/katenary).
- The second things is to help us to fix issues. If you're a Go developper, or if you want to fix the documentation,
your help is greatly appreciated.
- And then, of course, we need money, or sponsors.

### If you're a company

We will be happy to communicate your help by putting your logo on the website and in the documentaiton. You can sponsor
us by giving us some money, or by giving us some time of your developers, or leaving us some time to work on the project.

### If you're an individual

All donators will be listed on the website and in the documentation. You can give us some money by using
the [GitHub Sponsors]()

All main contributors[^3] will be listed on the website and in the documentation.

> If you want to be anonymous, please tell us.

[^3]: Main contributors are the people who have made a significant contribution to the project. It could be code,
documentation, or any other help. There is no defined rules, at this time, to evaluate the contribution.
It's a subjective decision.
