We have developed several practical examples demonstrating our CLI tool in action to help you quickly understand and implement the existing commands.

# Creating Workloads

To create a workload, you need to specify a payload file. This json file will consist of the cluster name you want to schedule the workload into and the kubernetes job payload you would like to schedule.

```bash
cedana-cli create workload --payload sample-workload.json
```
The below sample-workload.json can be used to test out the above command.

```json
{ "cluster_name": "<your_cluster_name>",
  "workload": {
  "apiVersion": "batch/v1",
  "kind": "Job",
  "metadata": {
    "name": "sleep-job-1",
    "namespace": "cedana",
    "labels": {
    "kueue.x-k8s.io/queue-name": "user-queue"
    }
  },
  "spec": {
    "template": {
      "spec": {
        "restartPolicy": "Never",
        "containers": [
          {
            "name": "sleep",
            "image": "busybox",
            "resources": {
              "requests": {
                "cpu": "0.5",
                "memory": "1Gi"
              },
              "limits": {
                "cpu": "32",
                "memory": "32Gi"
              }
            },
            "command": ["sleep", "infinity"]
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
cedana-cli delete workload --payload sample-workload.json
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
