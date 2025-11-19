# AWS 하이브리드 아키텍처
# Hot Path: ECS Fargate
# Cold Path: Lambda
# Data Layer: MSK, ElastiCache, RDS, S3

terraform {
  required_version = ">= 1.6"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  backend "s3" {
    bucket         = "aipx-terraform-state"
    key            = "aws/terraform.tfstate"
    region         = "ap-northeast-2"
    encrypt        = true
    dynamodb_table = "aipx-terraform-locks"
  }
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = "AIPX"
      Environment = var.environment
      ManagedBy   = "Terraform"
    }
  }
}

# Variables
variable "aws_region" {
  description = "AWS Region"
  type        = string
  default     = "ap-northeast-2"
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "vpc_cidr" {
  description = "VPC CIDR block"
  type        = string
  default     = "10.0.0.0/16"
}

# VPC Module
module "vpc" {
  source = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"

  name = "aipx-vpc-${var.environment}"
  cidr = var.vpc_cidr

  azs             = ["${var.aws_region}a", "${var.aws_region}b", "${var.aws_region}c"]
  private_subnets = ["10.0.10.0/24", "10.0.11.0/24", "10.0.12.0/24"]
  public_subnets  = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]

  enable_nat_gateway   = true
  single_nat_gateway   = var.environment == "dev" ? true : false
  enable_dns_hostnames = true
  enable_dns_support   = true
}

# Hot Path - ECS Fargate
module "hot_path" {
  source = "../modules/hot-path"

  provider_type      = "aws"
  environment        = var.environment
  vpc_id             = module.vpc.vpc_id
  private_subnet_ids = module.vpc.private_subnets
  kafka_brokers      = module.kafka.bootstrap_brokers
  redis_endpoint     = module.redis.primary_endpoint_address
}

# Cold Path - Lambda
module "cold_path" {
  source = "../modules/cold-path"

  provider_type    = "aws"
  environment      = var.environment
  kafka_topic_arn  = module.kafka.cluster_arn
  oms_endpoint     = module.hot_path.oms_endpoint
  storage_bucket   = module.storage.bucket_id
}

# Data Layer - MSK
module "kafka" {
  source = "../modules/data-layer/kafka/aws"

  environment        = var.environment
  vpc_id             = module.vpc.vpc_id
  private_subnet_ids = module.vpc.private_subnets
}

# Data Layer - ElastiCache
module "redis" {
  source = "../modules/data-layer/redis/aws"

  environment        = var.environment
  vpc_id             = module.vpc.vpc_id
  private_subnet_ids = module.vpc.private_subnets
}

# Data Layer - RDS PostgreSQL
module "database" {
  source = "../modules/data-layer/database/aws"

  environment        = var.environment
  vpc_id             = module.vpc.vpc_id
  private_subnet_ids = module.vpc.private_subnets
}

# Data Layer - S3
module "storage" {
  source = "../modules/data-layer/storage/aws"

  environment = var.environment
}

# Outputs
output "vpc_id" {
  value = module.vpc.vpc_id
}

output "kafka_brokers" {
  value = module.kafka.bootstrap_brokers
}

output "redis_endpoint" {
  value = module.redis.primary_endpoint_address
}

output "database_endpoint" {
  value = module.database.endpoint
}

output "oms_grpc_endpoint" {
  value = module.hot_path.oms_endpoint
}

output "cognitive_api_url" {
  value = module.cold_path.cognitive_api_endpoint
}
