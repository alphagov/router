variable "govuk_environment" {}

provider "aws" {
  region = "eu-west-1"
  default_tags {
    tags = {
      product              = "govuk"
      system               = "govuk-platform-engineering"
      service              = "router"
      environment          = var.govuk_environment
      owner                = "platform-engineering@digital.cabinet-office.gov.uk"
      repository           = "govuk-infrastructure"
    }
  }
}

data "tfe_outputs" "cluster_infrastructure" {
  organization = "govuk"
  workspace    = "cluster-infrastructure-${var.govuk_environment}"
}


resource "aws_s3_bucket" "bucket" {
  bucket = "govuk-router-routes-${var.govuk_environment}"
  force_destroy = true
}

data "aws_caller_identity" "current" {}

data "aws_iam_policy_document" "router_assume" {
  statement {
    actions = ["sts:AssumeRoleWithWebIdentity"]
    principals {
      type        = "Federated"
      identifiers = [data.tfe_outputs.cluster_infrastructure.nonsensitive_values.cluster_oidc_provider_arn]
    }
    condition {
      test     = "StringEquals"
      variable = "${replace(data.tfe_outputs.cluster_infrastructure.nonsensitive_values.cluster_oidc_provider_arn, "/^(.*provider/)/", "")}:sub"
      values   = ["system:serviceaccount:apps:router-routes-load"]
    }
    condition {
      test     = "StringEquals"
      variable = "${replace(data.tfe_outputs.cluster_infrastructure.nonsensitive_values.cluster_oidc_provider_arn, "/^(.*provider/)/", "")}:aud"
      values   = ["sts.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "router" {
  name = "router-routes-load-${var.govuk_environment}"
  assume_role_policy = data.aws_iam_policy_document.router_assume.json
}

data "aws_iam_policy_document" "router" {
  statement {
    actions = ["s3:*"]
    effect = "Allow"
    resources = [
      aws_s3_bucket.bucket.arn,
      "${aws_s3_bucket.bucket.arn}/*"
    ]
  }
}

resource "aws_iam_policy" "router" {
  name = "router-routes-load-${var.govuk_environment}"
  description = "Role to allow Router flat file loading and export"

  policy = data.aws_iam_policy_document.router.json
}

resource "aws_iam_role_policy_attachment" "router" {
  role = aws_iam_role.router.name
  policy_arn = aws_iam_policy.router.arn
}
