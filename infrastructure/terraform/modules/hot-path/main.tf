# Hot Path Module - ECS Fargate 기반 저지연 서비스
#
# 이 모듈은 다음을 포함합니다:
# - Data Ingestion Service (WebSocket 연결 유지)
# - Order Management Service (gRPC, 초저지연)

variable "provider_type" {
  description = "Cloud provider type (aws only)"
  type        = string
  default     = "aws"
}

variable "environment" {
  description = "Environment: dev, staging, production"
  type        = string
}

variable "vpc_id" {
  description = "VPC ID for networking"
  type        = string
}

variable "private_subnet_ids" {
  description = "Private subnet IDs for ECS tasks"
  type        = list(string)
}

variable "kafka_brokers" {
  description = "Kafka bootstrap servers"
  type        = string
}

variable "redis_endpoint" {
  description = "Redis endpoint"
  type        = string
}

# AWS ECS Fargate resources
module "aws_hot_path" {
  source = "./aws"

  environment        = var.environment
  vpc_id             = var.vpc_id
  private_subnet_ids = var.private_subnet_ids
  kafka_brokers      = var.kafka_brokers
  redis_endpoint     = var.redis_endpoint
}

output "ingestion_endpoint" {
  description = "Data Ingestion service endpoint"
  value       = module.aws_hot_path.ingestion_endpoint
}

output "oms_endpoint" {
  description = "Order Management Service gRPC endpoint"
  value       = module.aws_hot_path.oms_grpc_endpoint
}
