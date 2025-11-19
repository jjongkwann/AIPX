# Development Environment - AWS

environment = "dev"
aws_region  = "ap-northeast-2"

# VPC
vpc_cidr = "10.0.0.0/16"

# ECS Fargate 최소 스펙 (비용 절감)
ingestion_cpu    = 256
ingestion_memory = 512

oms_cpu    = 512
oms_memory = 1024

# MSK 개발용 작은 인스턴스
kafka_instance_type = "kafka.t3.small"
kafka_broker_count  = 1

# ElastiCache 최소 스펙
redis_node_type  = "cache.t3.micro"
redis_num_nodes  = 1

# RDS 개발용
database_instance_class = "db.t3.micro"
database_allocated_storage = 20
database_multi_az = false

# Lambda 기본 설정
lambda_memory_size = 512
lambda_timeout     = 60
