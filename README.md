# cedana-cli

Cedana is a framework for the democritization and (eventually) commodification of compute. We achieve this by leveraging checkpoint/restore to seamlessly migrate work across machines, clouds and beyond.

This repo contains a self serve CLI tool to allow developers to experiment with our system. With it, you can:

- Launch instances anywhere, with guaranteed price and capacity optimization. We look across your configured providers (AWS, Paperspace, etc.) to select the optimal instance defined in a provided job spec. This abstracts away cloud infra burdens.
- Leverage our client code (installed on every launched instance) to checkpoint/restore across instances, unlocking reliability, increased performance gains and decreased price.
- Deploy and manage any kind of job, whether a pyTorch training job, a webservice or a multibody physics simulation.

To access our managed service, contact founders@cedana.ai

## Usage
Cedana consists of the client code (found [here](https://github.com/nravic/cedana)) running on compute in the cloud (or anywhere else) and the orchestration/daemon, which runs on your local machine. 

To build from source: 
`go build`

To run: 
`./cedana-cli`

If you prefer to install from a package manager, we push to packagecloud and have a homebrew tap. Check out the [documentation](cedna.rtfd.io) for instructions. 

## Documentation
You can view the official documentation [here](cedana.rtfd.io). 

## Demo

https://github.com/cedana/cedana-cli/assets/409327/c5d06ca6-c200-4838-b2f0-6e780a49c7d4

(Note: The video is sped up for brevity to show how a CPU-bound PyTorch training job can be paused/migrated/resumed). 


## Todos 
We're working on building out a public roadmap. Until then, here's a few of the highest priority todos: 

- Add more cloud providers to arbitrage between
- `runc` container checkpointing
- Advanced optimizaiton strategies to pick and migrate work between clouds
- Way more tests
- GPU checkpointing 
- Simulation environment for rapid checkpoint/migrate 
- Kubernetes and cluster formation support
- Batch compute paradigms

For checkpoint/restore specific work, refer to the README in the client code repo.

## Contributing

See CONTRIBUTING.md for guidelines. 
