# cedana-cli

[Cedana](https://cedana.ai) is a framework for the democritization and (eventually) commodification of compute. We achieve this by leveraging checkpoint/restore to seamlessly migrate work across machines, clouds and beyond.

This repo contains a CLI tool to allow developers to experiment with our system. With it, you can:

- Launch instances anywhere, with guaranteed price and capacity optimization. We look across your configured providers (AWS, Paperspace, etc.) to select the optimal instance defined in a provided job spec. This abstracts away cloud infra burdens.
- Deploy and manage any kind of job, whether a pyTorch training job, a webservice or a multibody physics simulation.

Our managed system layers many more capabilities on top of this, such as: lifecycle management, policy systems, auto migration (through our novel checkpointing system (see [here](https://github.com/cedana/cedana))) and much more.

To access our managed service, contact <founders@cedana.ai>.

## Usage

To build from source:
`go build`

To run:
`./cedana-cli`

If you prefer to install from a package manager, we push to packagecloud and have a homebrew tap. Check out the [documentation](https://cedna.rtfd.io) for instructions.

## Documentation

You can view the official documentation [here](https://docs.cedana.ai).

## Deprecation Notice

`cedana-cli` used to have a self-serve tool, but it has been retired in favor of fulltime development on our managed platform. If you still wish to use it however, you can revert to previous versions (<=v0.2.8).

## Contributing

See CONTRIBUTING.md for guidelines.