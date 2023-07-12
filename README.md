# cedana-cli

Cedana is a framework for the democritization and (eventually) commodification of compute. We achieve this by leveraging checkpoint/restore to seamlessly migrate work across machines, clouds and beyond.

This repo is a self serve CLI tool to allow developers to experiment with our system. With it, you can:

- Launch instances anywhere, with guaranteed price and capacity optimization. We look across your configured providers (AWS, Paperspace, etc.) to select the optimal instance defined in a provided job spec. This abstracts away cloud infra burdens.
- Leverage our client code (installed on every launched instance) to checkpoint/restore across instances, unlocking reliability, increased performance gains and decreased price.
- Deploy any kind of job, whether a pyTorch training job or a webservice.

## Demo
