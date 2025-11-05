# How to configure router to serve routes from a flat file

Router has the ability to load routes from a JSON Lines file instead of the
Content Store database. This guide describes the end-to-end process of configuring
Router to serve routes from a file fetched from S3 in production.

It involves:

* Creating an S3 bucket via a local Terraform deployment
* Creating a Kubernetes service account that allows access to the S3 bucket
* Running a Kubernetes job to export routes from Content Store to the bucket
* Configuring Router's deployment to fetch the routes file from S3
* Setting the `ROUTER_ROUTES_FILE` env var to make Router load routes from the file

## Prerequisites

* `GOVUK_ENVIRONMENT` environment variable set to the environment name (e.g. `export GOVUK_ENVIRONMENT=integration`)
* Assume the `fulladmin` role in the AWS in the environment you're targeting:
  `eval $(gds aws govuk-${GOVUK_ENVIRONMENT}-fulladmin -e --art 8h)`
* AWS and Kubernetes cluster access configured in your terminal (`aws eks update-kubeconfig --name govuk`)
* Terraform installed and logged in to Terraform Cloud (`terraform login`)

Make sure you have assumed the role in your target environment before continuing.

## Create Required AWS Infrastructure

Initialise and apply the provided [Terraform infrastructure](https://github.com/alphagov/router/blob/main/flat-file-infrastructure/terraform/):

```bash
cd flat-file-infrastructure/terraform
terraform init -upgrade
terraform apply -var="govuk_environment=${GOVUK_ENVIRONMENT}"
```

This creates:
- S3 bucket: `govuk-router-routes-$GOVUK_ENVIRONMENT`
- IAM role: `router-routes-load-$GOVUK_ENVIRONMENT`
- Service account binding for `system:serviceaccount:apps:router-routes-load`

## Create Kubernetes service account

A service account is required for router to access the S3 bucket.
Fill in the AWS account ID and environment name in [`serviceaccount.yaml`](https://github.com/alphagov/router/blob/main/flat-file-infrastructure/kubernetes/serviceaccount.yaml) and apply it:

```sh
cd flat-file-infrastructure/kubernetes
# Get AWS account ID
export AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
# Fill in environment name and account ID placeholders
sed -e "s/<environment>/${GOVUK_ENVIRONMENT}/g" \
  -e "s/<aws-account-id>/${AWS_ACCOUNT_ID}/g" \
  serviceaccount.yaml > my-serviceaccount.yaml
# Create service account in cluster
kubectl apply -f my-serviceaccount.yaml
```

## Export Routes and Upload to S3

Routes are exported via a Kubernetes Job that runs the router in export mode and uploads the output to S3.

Fill in the environment name and [current production router version](https://github.com/alphagov/govuk-helm-charts/blob/main/charts/app-config/image-tags/production/router):

```sh
# Set router version variable
# Replace the value with the current production router version
export ROUTER_VERSION="v183"
# Replace placeholder environment name and router version
sed -e "s/<environment>/${GOVUK_ENVIRONMENT}/g" \
  -e "s/<router-version>/${ROUTER_VERSION}/g" \
  export-job.yaml > my-job.yaml
```

Create the job:

```sh
kubectl apply -f my-job.yaml
```

Monitor the job:

```sh
# Check job status
kubectl get jobs -n apps router-route-export

# View logs from export step
kubectl logs -n apps job/router-route-export -c router

# View logs from upload step
kubectl logs -n apps job/router-route-export -c upload
```

Clean up the job after completion:

```sh
kubectl delete -f my-job.yaml
```

The exported routes file will be available at `s3://govuk-router-routes-$GOVUK_ENVIRONMENT/routes.jsonl`.

## Configure Router to Load Routes from File

Update the Router app configuration in the corresponding environment's values file in [govuk-helm-charts](https://github.com/alphagov/govuk-helm-charts/tree/main/charts/app-config):

```yaml
- name: router
  ...
  # Add an initContainer which fetches the routes file from S3
  initContainers:
    - name: fetch-routes
      image: 172025368201.dkr.ecr.eu-west-1.amazonaws.com/github/alphagov/govuk/govuk-toolbox-image:v6
      # Fill in environment name here
      command: ["sh", "-c", "HOME=/tmp aws s3 cp s3://govuk-router-routes-<!!environment!!>/routes.jsonl /tmp/routes.jsonl"]
      volumeMounts:
        - name: app-tmp
          mountPath: /tmp
      securityContext:
        allowPrivilegeEscalation: false
        readOnlyRootFilesystem: true
        capabilities:
          drop: ["ALL"]
  ...
  # Set ROUTER_ROUTES_FILE env var
  extraEnv:
    - name: ROUTER_ROUTES_FILE
      value: /tmp/routes.jsonl
  ...
  # Set service account
  serviceAccount:
    enabled: true
    create: false
    # Use the same service account you created earlier
    name: router-routes-load
```

## Reverting Back to Content Store

To revert back to loading routes from PostgreSQL:

1. Revert changes made to govuk-helm-charts
2. Remove Kubernetes service account with `kubectl delete -f my-serviceaccount.yaml`
3. Empty S3 bucket: `aws s3 rm s3://govuk-router-routes-${GOVUK_ENVIRONMENT}/routes.jsonl`
4. Perform a `terraform apply -destroy -var="govuk_environment=${GOVUK_ENVIRONMENT}"` to remove AWS infrastructure
