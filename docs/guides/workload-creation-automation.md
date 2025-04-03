# Running 200+ Workloads with Cedana CLI

## Overview

This documentation explains how to use the create-workload.sh script to effectively submit and manage large batches of molecular dynamics simulation workloads to Cedana from pdb files stored in AWS S3 storage.

## Prerequisites

Before running the script, ensure you have the following:

1. AWS CLI installed and configured with `aws configure` to access the specified S3 bucket
2. Cedana CLI must be installed 
3. A valid workload template file (`workload.yml`) containing placeholders (`WORKING_DIR` and `JOB_NAME`) in the same directory as the script
4. Proper directory structure in your S3 bucket with molecular dynamics simulation files

## Script Configuration

```
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
