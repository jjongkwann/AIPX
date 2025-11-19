# Cold Path Module - Lambda 기반 서버리스 서비스
#
# 이 모듈은 다음을 포함합니다:
# - Strategy Worker (MSK trigger)
# - Cognitive Service (API Gateway + Lambda)
# - Notification Service (EventBridge trigger)
# - Data Recorder (MSK trigger, S3 저장)

variable "provider_type" {
  description = "Cloud provider type (aws only)"
  type        = string
  default     = "aws"
}

variable "environment" {
  description = "Environment: dev, staging, production"
  type        = string
}

variable "kafka_topic_arn" {
  description = "MSK cluster ARN for Lambda triggers"
  type        = string
}

variable "oms_endpoint" {
  description = "OMS gRPC endpoint for order submission"
  type        = string
}

variable "storage_bucket" {
  description = "S3 bucket for data storage"
  type        = string
}

# AWS Lambda functions
module "aws_cold_path" {
  source = "./aws"

  environment     = var.environment
  msk_cluster_arn = var.kafka_topic_arn
  oms_endpoint    = var.oms_endpoint
  s3_bucket       = var.storage_bucket
}

output "strategy_worker_name" {
  description = "Strategy Worker Lambda function ARN"
  value       = module.aws_cold_path.strategy_worker_arn
}

output "cognitive_api_endpoint" {
  description = "Cognitive Service API Gateway endpoint"
  value       = module.aws_cold_path.api_gateway_url
}
