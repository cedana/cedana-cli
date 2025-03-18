We've built some examples demonstrating scheduling workloads to help you quickly understand and implement our CLI tool. 

# Creating Workloads

## Running GROMACS

To create a workload, you need to specify a payload file. This json file will consist of the cluster name you want to schedule the workload into and the kubernetes job payload you would like to schedule.

```bash
cedana-cli create workload --payload simulation-workload.json
```
The below simulation-workload.json can be used to test out the above command.

```json
{ "cluster_name": "<your cluster name>",
  "workload": {
  "apiVersion": "batch/v1",
  "kind": "Job",
  "metadata": {
    "name": "gromacs-md-simulation",
    "namespace": "cedana",
    "labels": {
    "kueue.x-k8s.io/queue-name": "user-queue"
    }
  },
  "spec": {
    "template": {
      "spec": {
        "restartPolicy": "Never",
        "volumes": [
          {
            "name": "storage",
            "persistentVolumeClaim": {
              "claimName": "s3-simulation-pvc"
            }
          }
        ],
        "containers": [
          {
            "name": "gromacs",
            "image": "gromacs/gromacs:latest",
            "volumeMounts": [
              {
                "name": "storage",
                "mountPath": "/data"
              }
            ],
            "resources": {
              "requests": {
                "cpu": "8",
                "memory": "4Gi"
              },
              "limits": {
                "cpu": "32",
                "memory": "32Gi"
              }
            },
            "command": ["/bin/bash", "-c"],
            "args": [
              "set -ex; gmx pdb2gmx -f /data/complex.pdb -o complex_processed.gro -ff amber99sb -water tip3p; gmx editconf -f complex_processed.gro -o complex_newbox.gro -bt dodecahedron -d 1.0; echo \"17\" | gmx solvate -cp complex_newbox.gro -cs spc216.gro -o complex_solv.gro -p topol.top; gmx grompp -f /data/ions.mdp -c complex_solv.gro -p topol.top -o ions.tpr; echo \"13\" | gmx genion -s ions.tpr -o complex_solv_ions.gro -p topol.top -pname NA -nname CL -neutral;....."   ]
          }
        ]
      }
    }
  }
}
}
```

# Deleting Workloads

To delete a workload, use the same payload as specified in create. 

```bash
cedana-cli delete workload --payload simulation-workload.json
```

# Listing Workloads

Let's make sure that we have a cluster running to schedule new workloads into.

```bash
cedana-cli list cluster
```

Workloads spawn pods which can be listed through the following command:

```bash
cedana-cli list pod --cluster <your-cluster-name> --namespace cedana
```

All workloads are created under the cedana namespace, and have their lifecycles managed in the background. 
