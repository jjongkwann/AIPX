# AWS 하이브리드 서버리스 아키텍처
# Hot Path: ECS Fargate (Data Ingestion, OMS)
# Cold Path: Lambda (Strategy Worker, Cognitive, Notification, Recorder)
# Data Layer: MSK, ElastiCache, RDS, S3

# ============================================================================
# VPC Module
# ============================================================================
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"

  name = "aipx-vpc-${var.environment}"
  cidr = var.vpc_cidr

  azs             = ["${var.aws_region}a", "${var.aws_region}b", "${var.aws_region}c"]
  private_subnets = ["10.0.10.0/24", "10.0.11.0/24", "10.0.12.0/24"]
  public_subnets  = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]

  # NAT Gateway 설정 (dev: 단일, prod: 다중)
  enable_nat_gateway   = true
  single_nat_gateway   = var.environment == "dev" ? true : false
  enable_dns_hostnames = true
  enable_dns_support   = true

  # VPC Flow Logs
  enable_flow_log                      = true
  create_flow_log_cloudwatch_iam_role  = true
  create_flow_log_cloudwatch_log_group = true

  tags = {
    Name = "aipx-vpc-${var.environment}"
  }
}

# ============================================================================
# Data Layer - Amazon MSK (Kafka)
# ============================================================================
module "kafka" {
  source = "../modules/data-layer/kafka/aws"

  environment        = var.environment
  vpc_id             = module.vpc.vpc_id
  private_subnet_ids = module.vpc.private_subnets
  instance_type      = var.kafka_instance_type
  broker_count       = var.kafka_broker_count
}

# ============================================================================
# Data Layer - ElastiCache Redis
# ============================================================================
module "redis" {
  source = "../modules/data-layer/redis/aws"

  environment        = var.environment
  vpc_id             = module.vpc.vpc_id
  private_subnet_ids = module.vpc.private_subnets
  node_type          = var.redis_node_type
  num_cache_nodes    = var.redis_num_nodes
}

# ============================================================================
# Data Layer - RDS PostgreSQL
# ============================================================================
module "database" {
  source = "../modules/data-layer/database/aws"

  environment        = var.environment
  vpc_id             = module.vpc.vpc_id
  private_subnet_ids = module.vpc.private_subnets
  instance_class     = var.database_instance_class
  allocated_storage  = var.database_allocated_storage
  multi_az           = var.database_multi_az
}

# ============================================================================
# Data Layer - S3 Data Lake
# ============================================================================
module "storage" {
  source = "../modules/data-layer/storage/aws"

  environment = var.environment
}

# ============================================================================
# Hot Path - ECS Fargate
# ============================================================================
module "hot_path" {
  source = "../modules/hot-path"

  provider_type      = "aws"
  environment        = var.environment
  vpc_id             = module.vpc.vpc_id
  private_subnet_ids = module.vpc.private_subnets
  public_subnet_ids  = module.vpc.public_subnets
  kafka_brokers      = module.kafka.bootstrap_brokers
  redis_endpoint     = module.redis.primary_endpoint_address
  database_endpoint  = module.database.endpoint

  # ECS Fargate 리소스 설정
  ingestion_cpu    = var.ingestion_cpu
  ingestion_memory = var.ingestion_memory
  oms_cpu          = var.oms_cpu
  oms_memory       = var.oms_memory
}

# ============================================================================
# Cold Path - Lambda Functions
# ============================================================================
module "cold_path" {
  source = "../modules/cold-path"

  provider_type    = "aws"
  environment      = var.environment
  vpc_id           = module.vpc.vpc_id
  private_subnet_ids = module.vpc.private_subnets
  kafka_cluster_arn  = module.kafka.cluster_arn
  kafka_brokers      = module.kafka.bootstrap_brokers
  oms_endpoint       = module.hot_path.oms_endpoint
  storage_bucket     = module.storage.bucket_id
  database_endpoint  = module.database.endpoint
  redis_endpoint     = module.redis.primary_endpoint_address

  # Lambda 설정
  memory_size = var.lambda_memory_size
  timeout     = var.lambda_timeout
}
