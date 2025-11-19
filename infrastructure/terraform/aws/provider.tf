# Terraform 및 Provider 버전
terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# AWS Provider 설정
provider "aws" {
  region = var.aws_region

  # 모든 리소스에 기본 태그 적용
  default_tags {
    tags = merge(
      {
        Project     = "AIPX"
        Environment = var.environment
        ManagedBy   = "Terraform"
      },
      var.tags
    )
  }
}
