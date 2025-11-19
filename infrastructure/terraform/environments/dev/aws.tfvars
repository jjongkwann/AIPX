# AWS 개발 환경 변수
# Region
aws_region = "ap-northeast-2"

# Environment
environment = "dev"

# Network
vpc_cidr = "10.0.0.0/16"

# ============================================================================
# ECS Fargate 설정 (Hot Path)
# ============================================================================
# Data Ingestion Service (WebSocket) - 최소 스펙
ingestion_cpu    = 256  # 0.25 vCPU
ingestion_memory = 512  # 512 MB

# OMS (gRPC) - 최소 스펙
oms_cpu    = 512   # 0.5 vCPU
oms_memory = 1024  # 1 GB

# ============================================================================
# Amazon MSK (Kafka) 설정
# ============================================================================
kafka_instance_type = "kafka.t3.small"  # Dev 환경용 소형 인스턴스
kafka_broker_count  = 1                 # Dev는 단일 브로커로 비용 절감

# ============================================================================
# ElastiCache Redis 설정
# ============================================================================
redis_node_type = "cache.t3.micro"  # Dev 환경용 최소 스펙
redis_num_nodes = 1                 # Dev는 단일 노드

# ============================================================================
# RDS PostgreSQL 설정
# ============================================================================
database_instance_class     = "db.t3.micro"  # Dev 환경용 최소 스펙
database_allocated_storage  = 20             # 20 GB
database_multi_az           = false          # Dev는 단일 AZ로 비용 절감

# ============================================================================
# Lambda 설정 (Cold Path)
# ============================================================================
lambda_memory_size = 512  # 512 MB (Dev 환경용 최소 스펙)
lambda_timeout     = 60   # 1분

# ============================================================================
# 공통 태그
# ============================================================================
tags = {
  CostCenter = "Development"
  Owner      = "DevOps Team"
  Purpose    = "AIPX Development Environment"
}
