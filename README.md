# mTLS Secret Updater

`mtls-secret-updater` is a tool that goes hand-in-hand with [`spiffe-helper`](https://github.com/spiffe/spiffe-helper) to deliver SPIFFE X.509 SVIDs and trust bundles to applications running in Kubernetes that are set up to consume secrets (and reloaded on secret update).

## Use

An example deployment is given in `deployment/manifests/test-pod.yaml`.

## Obtain

Pre-built `mtls-secret-updater` images may be downloaded from the project's [GitHub releases](https://github.com/cofide/mtls-secret-updater/releases) page.
Alternatively, `mtls-secret-updater` image may be built from source code.

## Build

Building a `mtls-secret-updater` image requires:

* [Docker](https://docs.docker.com/engine/install/)
* [`just`](https://github.com/casey/just) as a command runner

To build the `mtls-secret-updater` image, run:

```sh
just build
```

## Production use cases

<div style="float: left; margin-right: 10px;">
    <a href="https://www.cofide.io">
        <img src="docs/img/cofide-colour-blue.svg" width="40" alt="Cofide">
    </a>
</div>

`mtls-secret-updater` is a project developed and maintained by [Cofide](https://www.cofide.io). We're building a workload identity platform that is seamless and secure for multi and hybrid cloud environments. If you have a production use case with need for greater flexibility, control and visibility, with enterprise-level support, please [speak with us](mailto:hello@cofide.io) to find out more about the [Cofide](https://www.cofide.io) early access programme 👀.
