# cedana-cli

Cedana is a framework for the democritization and (eventually) commodification of compute. We achieve this by leveraging checkpoint/restore to seamlessly migrate work across machines, clouds and beyond.

This repo is a self serve CLI tool to allow developers to experiment with our system. With it, you can:

- Launch instances anywhere, with guaranteed price and capacity optimization. We look across your configured providers (AWS, Paperspace, etc.) to select the optimal instance defined in a provided job spec. This abstracts away cloud infra burdens.
- Leverage our client code (installed on every launched instance) to checkpoint/restore across instances, unlocking reliability, increased performance gains and decreased price.
- Deploy any kind of job, whether a pyTorch training job or a webservice.

## Usage 


## Demo
https://github.com/cedana/cedana-cli/assets/409327/c5d06ca6-c200-4838-b2f0-6e780a49c7d4

(Note: The video is sped up for brevity to show how a CPU-bound PyTorch training job can be paused/migrated/resumed). 
