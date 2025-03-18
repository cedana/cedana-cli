We have developed several practical examples demonstrating our CLI tool in action to help you quickly understand and implement the existing commands.

# Creating Workloads

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
              "set -ex; gmx pdb2gmx -f /data/complex.pdb -o complex_processed.gro -ff amber99sb -water tip3p; gmx editconf -f complex_processed.gro -o complex_newbox.gro -bt dodecahedron -d 1.0; echo \"17\" | gmx solvate -cp complex_newbox.gro -cs spc216.gro -o complex_solv.gro -p topol.top; gmx grompp -f /data/ions.mdp -c complex_solv.gro -p topol.top -o ions.tpr; echo \"13\" | gmx genion -s ions.tpr -o complex_solv_ions.gro -p topol.top -pname NA -nname CL -neutral; gmx editconf -f complex_solv_ions.gro -o complex_solv_ions_center.gro -c; gmx grompp -f /data/em.mdp -c complex_solv_ions_center.gro -p topol.top -o em.tpr; gmx mdrun -deffnm em; echo \"0\" | gmx trjconv -s em.tpr -f em.gro -o em_whole.gro -pbc whole; echo -e \"1\\n0\" | gmx trjconv -s em.tpr -f em_whole.gro -o em_cluster.gro -pbc cluster; echo -e \"1\\n0\" | gmx trjconv -s em.tpr -f em_cluster.gro -o em_centered.gro -center; gmx grompp -f /data/nvt.mdp -c em_centered.gro -r em_centered.gro -p topol.top -o nvt.tpr; gmx mdrun -deffnm nvt; echo \"0\" | gmx trjconv -s nvt.tpr -f nvt.gro -o nvt_whole.gro -pbc whole; echo -e \"1\\n0\" | gmx trjconv -s nvt.tpr -f nvt_whole.gro -o nvt_cluster.gro -pbc cluster; echo -e \"1\\n0\" | gmx trjconv -s nvt.tpr -f nvt_cluster.gro -o nvt_centered.gro -center; gmx grompp -f /data/npt.mdp -c nvt_centered.gro -r nvt_centered.gro -t nvt.cpt -p topol.top -o npt.tpr; gmx mdrun -deffnm npt; echo \"0\" | gmx trjconv -s npt.tpr -f npt.gro -o npt_whole.gro -pbc whole; echo -e \"1\\n0\" | gmx trjconv -s npt.tpr -f npt_whole.gro -o npt_cluster.gro -pbc cluster; echo -e \"1\\n0\" | gmx trjconv -s npt.tpr -f npt_cluster.gro -o npt_centered.gro -center; gmx grompp -f /data/md.mdp -c npt_centered.gro -t npt.cpt -p topol.top -o md.tpr; gmx mdrun -deffnm md"
            ]
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

All workloads are created under cedana namespace only.
