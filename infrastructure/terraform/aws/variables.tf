# AWS 리전
variable "aws_region" {
  description = "AWS Region"
  type        = string
  default     = "ap-northeast-2"
}

# 환경
variable "environment" {
  description = "Environment name (dev, staging, production)"
  type        = string
}

# VPC CIDR
variable "vpc_cidr" {
  description = "VPC CIDR block"
  type        = string
  default     = "10.0.0.0/16"
}

# ECS Fargate 설정
variable "ingestion_cpu" {
  description = "Data Ingestion CPU units (256 = 0.25 vCPU)"
  type        = number
  default     = 512
}

variable "ingestion_memory" {
  description = "Data Ingestion memory in MB"
  type        = number
  default     = 1024
}

variable "oms_cpu" {
  description = "OMS CPU units"
  type        = number
  default     = 1024
}

variable "oms_memory" {
  description = "OMS memory in MB"
  type        = number
  default     = 2048
}

# MSK 설정
variable "kafka_instance_type" {
  description = "MSK broker instance type"
  type        = string
  default     = "kafka.m5.large"
}

variable "kafka_broker_count" {
  description = "Number of MSK brokers"
  type        = number
  default     = 3
}

# ElastiCache 설정
variable "redis_node_type" {
  description = "ElastiCache node type"
  type        = string
  default     = "cache.r6g.large"
}

variable "redis_num_nodes" {
  description = "Number of cache nodes"
  type        = number
  default     = 6
}

# RDS 설정
variable "database_instance_class" {
  description = "RDS instance class"
  type        = string
  default     = "db.r6g.xlarge"
}

variable "database_allocated_storage" {
  description = "RDS allocated storage in GB"
  type        = number
  default     = 100
}

variable "database_multi_az" {
  description = "Enable Multi-AZ deployment"
  type        = bool
  default     = true
}

# Lambda 설정
variable "lambda_memory_size" {
  description = "Lambda memory size in MB"
  type        = number
  default     = 2048
}

variable "lambda_timeout" {
  description = "Lambda timeout in seconds"
  type        = number
  default     = 300
}

# 태그
variable "tags" {
  description = "Common tags for all resources"
  type        = map(string)
  default     = {}
}
