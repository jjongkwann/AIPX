# VPC Outputs
output "vpc_id" {
  description = "VPC ID"
  value       = module.vpc.vpc_id
}

output "public_subnets" {
  description = "Public subnet IDs"
  value       = module.vpc.public_subnets
}

output "private_subnets" {
  description = "Private subnet IDs"
  value       = module.vpc.private_subnets
}

# Kafka Outputs
output "kafka_bootstrap_brokers" {
  description = "MSK bootstrap brokers"
  value       = module.kafka.bootstrap_brokers
  sensitive   = true
}

output "kafka_zookeeper_connect" {
  description = "MSK Zookeeper connection string"
  value       = module.kafka.zookeeper_connect_string
  sensitive   = true
}

# Redis Outputs
output "redis_endpoint" {
  description = "ElastiCache Redis primary endpoint"
  value       = module.redis.primary_endpoint_address
  sensitive   = true
}

output "redis_port" {
  description = "ElastiCache Redis port"
  value       = module.redis.port
}

# Database Outputs
output "database_endpoint" {
  description = "RDS PostgreSQL endpoint"
  value       = module.database.endpoint
  sensitive   = true
}

output "database_name" {
  description = "Database name"
  value       = module.database.db_name
}

# Hot Path Outputs
output "oms_grpc_endpoint" {
  description = "OMS gRPC endpoint (internal NLB)"
  value       = module.hot_path.oms_endpoint
}

output "ingestion_service_name" {
  description = "Data Ingestion ECS service name"
  value       = module.hot_path.ingestion_endpoint
}

# Cold Path Outputs
output "strategy_worker_arn" {
  description = "Strategy Worker Lambda ARN"
  value       = module.cold_path.strategy_worker_name
}

output "cognitive_api_url" {
  description = "Cognitive Service API Gateway URL"
  value       = module.cold_path.cognitive_api_endpoint
}

# S3 Outputs
output "data_lake_bucket" {
  description = "S3 Data Lake bucket name"
  value       = module.storage.bucket_id
}
