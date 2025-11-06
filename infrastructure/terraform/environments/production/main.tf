terraform {
  required_version = ">= 1.5"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  backend "s3" {
    bucket         = "immobilier-saas-terraform-state"
    key            = "production/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
    dynamodb_table = "immobilier-saas-terraform-lock"
  }
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = "immobilier-saas"
      Environment = "production"
      ManagedBy   = "Terraform"
    }
  }
}

# Local variables
locals {
  project_name = "immobilier-saas"
  environment  = "production"
  azs          = ["${var.aws_region}a", "${var.aws_region}b", "${var.aws_region}c"]
}

# VPC Module
module "vpc" {
  source = "../../modules/vpc"

  project_name       = local.project_name
  environment        = local.environment
  vpc_cidr           = "10.0.0.0/16"
  availability_zones = local.azs
  enable_nat_gateway = true
  enable_flow_logs   = true
}

# RDS PostgreSQL Module
module "rds" {
  source = "../../modules/rds"

  project_name            = local.project_name
  environment             = local.environment
  vpc_id                  = module.vpc.vpc_id
  private_subnet_ids      = module.vpc.private_subnet_ids
  allowed_security_groups = [module.ecs.ecs_security_group_id]

  instance_class          = "db.t3.medium"
  allocated_storage       = 100
  database_name           = "immobilier_saas"
  master_username         = var.db_username
  master_password         = var.db_password

  backup_retention_period      = 7
  multi_az                     = true
  deletion_protection          = true
  skip_final_snapshot          = false
  enable_performance_insights  = true

  create_read_replica     = true
  replica_instance_class  = "db.t3.medium"

  alarm_sns_topic_arn = aws_sns_topic.alerts.arn
}

# ElastiCache Redis Module
module "redis" {
  source = "../../modules/redis"

  project_name            = local.project_name
  environment             = local.environment
  vpc_id                  = module.vpc.vpc_id
  private_subnet_ids      = module.vpc.private_subnet_ids
  allowed_security_groups = [module.ecs.ecs_security_group_id]

  node_type               = "cache.t3.micro"
  num_cache_nodes         = 2
  engine_version          = "7.0"
  automatic_failover      = true

  alarm_sns_topic_arn = aws_sns_topic.alerts.arn
}

# S3 Bucket for uploads
module "s3" {
  source = "../../modules/s3"

  project_name    = local.project_name
  environment     = local.environment
  bucket_name     = "immobilier-saas-uploads-production"
  enable_versioning = true
  enable_lifecycle  = true

  cors_allowed_origins = ["https://www.immobilier.com", "https://app.immobilier.com"]
}

# Application Load Balancer
module "alb" {
  source = "../../modules/alb"

  project_name       = local.project_name
  environment        = local.environment
  vpc_id             = module.vpc.vpc_id
  public_subnet_ids  = module.vpc.public_subnet_ids

  certificate_arn    = var.acm_certificate_arn
  enable_waf         = true
}

# ECS Cluster and Services
module "ecs" {
  source = "../../modules/ecs"

  project_name       = local.project_name
  environment        = local.environment
  vpc_id             = module.vpc.vpc_id
  private_subnet_ids = module.vpc.private_subnet_ids
  alb_target_group_arn = module.alb.target_group_arn

  # API Service
  api_image           = var.api_image
  api_cpu             = 1024
  api_memory          = 2048
  api_desired_count   = 2
  api_min_capacity    = 2
  api_max_capacity    = 10

  # Environment variables
  api_environment_variables = {
    ENVIRONMENT  = "production"
    DATABASE_URL = "postgres://${var.db_username}:${var.db_password}@${module.rds.endpoint}/${module.rds.database_name}?sslmode=require"
    REDIS_URL    = "rediss://${module.redis.endpoint}:6379/0"
    S3_BUCKET    = module.s3.bucket_name
    S3_REGION    = var.aws_region
  }

  # Secrets (from AWS Secrets Manager)
  api_secrets = {
    JWT_PRIVATE_KEY     = var.jwt_private_key_secret_arn
    JWT_PUBLIC_KEY      = var.jwt_public_key_secret_arn
    WAVE_API_KEY        = var.wave_api_key_secret_arn
    TWILIO_AUTH_TOKEN   = var.twilio_auth_token_secret_arn
  }
}

# SNS Topic for Alerts
resource "aws_sns_topic" "alerts" {
  name = "${local.project_name}-alerts-${local.environment}"

  tags = {
    Name        = "${local.project_name}-alerts-${local.environment}"
    Environment = local.environment
  }
}

resource "aws_sns_topic_subscription" "alerts_email" {
  topic_arn = aws_sns_topic.alerts.arn
  protocol  = "email"
  endpoint  = var.alert_email
}

# CloudWatch Log Groups
resource "aws_cloudwatch_log_group" "api" {
  name              = "/ecs/${local.project_name}-api-${local.environment}"
  retention_in_days = 7

  tags = {
    Name        = "${local.project_name}-api-logs-${local.environment}"
    Environment = local.environment
  }
}

# Outputs
output "alb_dns_name" {
  description = "DNS name of the Application Load Balancer"
  value       = module.alb.alb_dns_name
}

output "rds_endpoint" {
  description = "RDS endpoint"
  value       = module.rds.endpoint
  sensitive   = true
}

output "redis_endpoint" {
  description = "Redis endpoint"
  value       = module.redis.endpoint
  sensitive   = true
}

output "s3_bucket_name" {
  description = "S3 bucket name"
  value       = module.s3.bucket_name
}
