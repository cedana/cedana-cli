# Scheduling 200+ GROMACS Workloads with Cedana 

## Overview

This documentation runs through an example using a script to effectively submit and manage large batches of molecular dynamics simulation workloads to Cedana from pdb files stored in AWS S3 storage.

## Prerequisites

Before running the script, ensure you have the following:

1. AWS CLI installed and configured with `aws configure` to access the specified S3 bucket
2. Cedana CLI must be installed 
3. A valid workload template file (`workload.yml`) containing placeholders (`WORKING_DIR` and `JOB_NAME`) in the same directory as the script
4. Proper directory structure in your S3 bucket with molecular dynamics simulation files

## Script Configuration

```bash
#!/bin/bash

BUCKET_NAME="your-bucket-name"
PREFIX="gromacs_test/"
WORKLOAD_CONFIG="./workload.yml"
TEMP_WORKLOAD_CONFIG="./tmp-workload.yml"

# Sanitize job names for Kubernetes
sanitize_job_name() {
    local name="md-simul-${1//_/-}"  # Replace _ with -
    name=$(echo "$name" | tr '[:upper:]' '[:lower:]')  # Convert to lowercase
    name=$(echo "$name" | sed 's/--*/-/g' | sed 's/-$//')  # Remove double hyphens & trailing hyphen
    echo "$name"
}

# Get folders from S3
FOLDERS=$(aws s3 ls "s3://$BUCKET_NAME/$PREFIX" --recursive | awk '{print $NF}' | awk -F'/' 'NF>2 {print $2}' | sort -u)

for folder in $FOLDERS; do
    echo "Processing folder: $folder"
    JOB_NAME=$(sanitize_job_name "$folder")
    # Create a temporary workload with right complex.pdb file by replacing placeholders
    sed -e "s|WORKING_DIR|$folder|g" -e "s|JOB_NAME|$JOB_NAME|g" "$WORKLOAD_CONFIG" > "$TEMP_WORKLOAD_CONFIG"
    # Submit job
    cedana-cli create workload --payload "$TEMP_WORKLOAD_CONFIG" --contentType yaml
    rm -f "$TEMP_WORKLOAD_CONFIG"
done
```

The script uses the following default values that you may need to adjust:

- `BUCKET_NAME="customer-simulation"`: The S3 bucket containing simulation data
- `PREFIX="gromacs_test/"`: The prefix path within the bucket to search for simulation folders
- `WORKLOAD_CONFIG="./workload.yml"`: Path to the template workload configuration

## Execution Process

Sample workload.yml file:

```yaml
cluster_name: your-eks-cluster
workload:
  apiVersion: batch/v1
  kind: Job
  metadata:
    name: JOB_NAME
    namespace: cedana
    labels:
      kueue.x-k8s.io/queue-name: user-queue
  spec:
    template:
      spec:
        restartPolicy: Never
        volumes:
          - name: storage
            persistentVolumeClaim:
              claimName: s3-simulation-pvc
        containers:
          - name: gromacs
            image: gromacs/gromacs:latest
            resources:
              requests:
                cpu: "16"
                memory: "16Gi"
              limits:
                cpu: "32"
                memory: "32Gi"
            volumeMounts:
              - name: storage
                mountPath: /data
            command: ["/bin/bash", "-c"]
            args:
              - |
                set -ex  # Exit on error and print commands
                cp -rf /data/gromacs_test /gromacs_test
                mkdir -p /gromacs_test
                cd /gromacs_test/WORKING_DIR
                gmx pdb2gmx -f "WORKING_DIR.pdb" -o prep_processed.gro -ff amber99sb -water tip3p
                gmx editconf -f prep_processed.gro -o prep_newbox.gro -bt dodecahedron -d 1.5 -c
                gmx solvate -cp prep_newbox.gro -cs spc216.gro -o prep_solv.gro -p topol.top
                gmx grompp -f ../ions.mdp -c prep_solv.gro -p topol.top -o ions.tpr
                echo "13" | gmx genion -s ions.tpr -o prep_solv_ions.gro -p topol.top -pname NA -nname CL -neutral
                gmx grompp -f ../em.mdp -c prep_solv_ions.gro -p topol.top -o em.tpr
                gmx mdrun -deffnm em
                echo -e "1\n0" | gmx trjconv -s em.tpr -f em.gro -o em_centered.gro -pbc mol -center
                gmx grompp -f ../nvt.mdp -c em_centered.gro -r em_centered.gro -p topol.top -o nvt.tpr
                gmx mdrun -deffnm nvt
                echo -e "1\n0" | gmx trjconv -s nvt.tpr -f nvt.gro -o nvt_centered.gro -pbc mol -center
                gmx grompp -f ../npt.mdp -c nvt_centered.gro -r nvt_centered.gro -t nvt.cpt -p topol.top -o npt.tpr
                cp -rf /gromacs_test /data/gromacs_output
                gmx mdrun -deffnm npt
                echo -e "1\n0" | gmx trjconv -s npt.tpr -f npt.gro -o npt_centered.gro -pbc mol -center
                python ../find_number.py
                {
                  read receptor_start
                  read receptor_end
                  read ligand_start
                  read ligand_end
                } < atom_number.txt
                gmx make_ndx -f npt_centered.gro -o index.ndx <<< "q"
                group_count=$(grep "\[" index.ndx | wc -l)
                echo -e "a ${receptor_start}-${receptor_end}\nname $((group_count)) receptor\na ${ligand_start}-${ligand_end}\nname $((group_count + 1)) ligand\nq" | gmx make_ndx -f npt_centered.gro -n index.ndx -o index.ndx
                gmx grompp -f ../md.mdp -c npt_centered.gro -t npt.cpt -p topol.top -n index.ndx -o md.tpr
                gmx mdrun -deffnm md
                echo -e "51\n52" | gmx energy -f md.edr -o interaction_energy.xvg
                cd ..
                cp -rf /gromacs_test/WORKING_DIR /data/gromacs_output/WORKING_DIR
```

To execute the script run:
   ```bash
   chmod +x create-workload.sh
   ./create-workload.sh
   ```
When run, the script will:

1. Connect to the specified S3 bucket and list all folders under the defined prefix
2. For each folder found:
   - Generate a workload name based on the folder name
   - Create a temporary workload configuration file with the appropriate substitutions
   - Submit the workload to Cedana using the Cedana CLI
   - Remove the temporary workload file

## Deleting Workloads

To delete previously scheduled workloads, use the same script with a modified command. Simply replace `create` with `delete` in the Cedana CLI command:

```bash
cedana-cli delete workload --payload "$TEMP_WORKLOAD_CONFIG" --contentType yaml
```

## Troubleshooting

- Ensure AWS CLI credentials are correctly configured
- Verify the S3 bucket name and prefix are accurate
- Confirm that the `workload.yml` template file contains the correct placeholders (`WORKING_DIR` and `JOB_NAME`)
- Check that Cedana CLI is properly installed and authenticated

## Additional Notes

The script sanitizes folder names for Kubernetes compatibility by:
- Converting underscores to hyphens
- Changing all characters to lowercase
- Removing consecutive hyphens and trailing hyphens
