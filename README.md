# cedana-cli

[Cedana](https://cedana.ai) is a framework for the democritization and (eventually) commodification of compute. We achieve this by leveraging checkpoint/restore to seamlessly migrate work across machines, clouds and beyond.

This repo contains a CLI tool to allow developers to experiment with our system.

## Usage

To build from source:
```bash
go build
# install on linux
install ./cedana-cli /usr/local/bin
```

To get started:
```bash
export CEDANA_URL="https://sandbox.cedana.ai/v1"
export CEDANA_AUTH_TOKEN=<Your auth token from https://auth.cedana.com>

cedana-cli --help
```

## Documentation

We are still working on the documentation.

## Deprecation Notice

`cedana-cli` used to have a self-serve tool, but it has been retired in favor of fulltime development on our managed platform. If you still wish to use it however, you can revert to previous versions (<=v0.2.8).

<details>
### Deprecated functionality description

With it, you can:

- Launch instances anywhere, with guaranteed price and capacity optimization. We look across your configured providers (AWS, Paperspace, etc.) to select the optimal instance defined in a provided job spec. This abstracts away cloud infra burdens. (On older versions only)
- Deploy and manage any kind of job, whether a pyTorch training job, a webservice or a multibody physics simulation on kubernetes.

Our managed system layers many more capabilities on top of this, such as: lifecycle management, policy systems, auto migration (through our novel checkpointing system (see [here](https://github.com/cedana/cedana))) and much more.

To access our managed service, contact <founders@cedana.ai>.
</details>

## Contributing

See CONTRIBUTING.md for guidelines.
