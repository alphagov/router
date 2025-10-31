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

* Full access to AWS in the environment you're targeting
* AWS and Kubernetes cluster access configured in your terminal
* Terraform installed

## Create Required AWS Infrastructure

Initialise and apply the provided [Terraform infrastructure](https://github.com/alphagov/router/blob/main/flat-file-infrastructure/terraform/):

```bash
cd flat-file-infrastructure/terraform
terraform init -upgrade
terraform apply -var="govuk_environment=<environment>"
```

This creates:
- S3 bucket: `govuk-router-routes-<environment>`
- IAM role: `router-routes-load-<environment>`
- Service account binding for `system:serviceaccount:apps:router-routes-load`

## Create Kubernetes service account

A service account is required for router to access the S3 bucket.
Fill in the AWS account ID and environment name in [`serviceaccount.yaml`](https://github.com/alphagov/router/blob/main/flat-file-infrastructure/kubernetes/serviceaccount.yaml) and apply it:

```sh
kubectl apply -f serviceaccount.yaml
```

## Export Routes and Upload to S3

Routes are exported via a Kubernetes Job that runs the router in export mode and uploads the output to S3.

Edit the [job file](https://github.com/alphagov/router/blob/main/flat-file-infrastructure/kubernetes/export-job.yaml) to set the following values before applying:
- Router image tag (line 23)
- S3 bucket name (line 44)

Create the job:

```sh
kubectl apply -f export-job.yaml
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
kubectl delete -f export-job.yaml
```

The exported routes file will be available at `s3://govuk-router-routes-<environment>/routes.jsonl`.

## Configure Router to Load Routes from File

Update the Router app configuration in the corresponding environment's values file in [govuk-helm-charts](https://github.com/alphagov/govuk-helm-charts/tree/main/charts/app-config):

```yaml
- name: router
  ...
  # Required for AWS CLI to function
  extraVolumes:
    - name: aws-tmp
      emptyDir: {}
  # Add an initContainer which fetches the routes file from S3
  initContainers:
    - name: fetch-routes
      image: 172025368201.dkr.ecr.eu-west-1.amazonaws.com/github/alphagov/govuk/govuk-toolbox-image:v6
      command: ["sh", "-c", "aws s3 cp s3://govuk-router-routes-<environment>/routes.jsonl /tmp/routes.jsonl"]
      volumeMounts:
        - name: app-tmp
          mountPath: /tmp
        - name: aws-tmp
          mountPath: /home/user/.aws
      securityContext:
        allowPrivilegeEscalation: false
        readOnlyRootFilesystem: true
        capabilities:
          drop: ["ALL"]
  ...
  # Set ROUTER_ROUTES_FILE env var
  extraEnv:
    - name: router
      env:
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
2. Remove Kubernetes service account with `kubectl delete -f serviceaccount.yaml`
3. Empty S3 bucket: `aws s3 rm s3://govuk-router-routes-<environment>/routes.jsonl`
4. Perform a `terraform apply -destroy -var="govuk_environment=<environment>"` to remove AWS infrastructure
