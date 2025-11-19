#!/bin/bash

# AIPX Terraform Backend Setup Script
# This script creates the S3 bucket and DynamoDB table for Terraform state management

set -e

REGION="ap-northeast-2"
BUCKET_NAME="aipx-terraform-state"
DYNAMODB_TABLE="aipx-terraform-locks"

echo "🚀 AIPX AWS Terraform Backend Setup"
echo "===================================="
echo ""

# Check if AWS CLI is installed
if ! command -v aws &> /dev/null; then
    echo "❌ AWS CLI is not installed. Please install it first."
    echo "   https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html"
    exit 1
fi

# Check if AWS credentials are configured
if ! aws sts get-caller-identity &> /dev/null; then
    echo "❌ AWS credentials are not configured."
    echo "   Run: aws configure"
    exit 1
fi

echo "✅ AWS CLI configured"
echo "   Account: $(aws sts get-caller-identity --query Account --output text)"
echo "   Region: $REGION"
echo ""

# Create S3 bucket
echo "📦 Creating S3 bucket: $BUCKET_NAME"
if aws s3 ls "s3://$BUCKET_NAME" 2>&1 | grep -q 'NoSuchBucket'; then
    aws s3 mb "s3://$BUCKET_NAME" --region "$REGION"
    echo "   ✅ S3 bucket created"
else
    echo "   ⚠️  S3 bucket already exists"
fi

# Enable versioning
echo "🔄 Enabling versioning on S3 bucket"
aws s3api put-bucket-versioning \
    --bucket "$BUCKET_NAME" \
    --versioning-configuration Status=Enabled \
    --region "$REGION"
echo "   ✅ Versioning enabled"

# Enable encryption
echo "🔒 Enabling encryption on S3 bucket"
aws s3api put-bucket-encryption \
    --bucket "$BUCKET_NAME" \
    --server-side-encryption-configuration '{
        "Rules": [{
            "ApplyServerSideEncryptionByDefault": {
                "SSEAlgorithm": "AES256"
            }
        }]
    }' \
    --region "$REGION"
echo "   ✅ Encryption enabled (AES256)"

# Block public access
echo "🚫 Blocking public access on S3 bucket"
aws s3api put-public-access-block \
    --bucket "$BUCKET_NAME" \
    --public-access-block-configuration \
        "BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true" \
    --region "$REGION"
echo "   ✅ Public access blocked"

# Create DynamoDB table
echo "🗄️  Creating DynamoDB table: $DYNAMODB_TABLE"
if aws dynamodb describe-table --table-name "$DYNAMODB_TABLE" --region "$REGION" &> /dev/null; then
    echo "   ⚠️  DynamoDB table already exists"
else
    aws dynamodb create-table \
        --table-name "$DYNAMODB_TABLE" \
        --attribute-definitions AttributeName=LockID,AttributeType=S \
        --key-schema AttributeName=LockID,KeyType=HASH \
        --billing-mode PAY_PER_REQUEST \
        --region "$REGION" \
        --tags Key=Project,Value=AIPX Key=Environment,Value=shared
    echo "   ✅ DynamoDB table created"

    # Wait for table to be active
    echo "   ⏳ Waiting for table to become active..."
    aws dynamodb wait table-exists --table-name "$DYNAMODB_TABLE" --region "$REGION"
    echo "   ✅ Table is active"
fi

echo ""
echo "✅ Terraform backend setup complete!"
echo ""
echo "📝 Next steps:"
echo "   1. cd infrastructure/terraform/aws"
echo "   2. terraform init"
echo "   3. terraform plan -var-file=../environments/dev/aws.tfvars"
echo "   4. terraform apply -var-file=../environments/dev/aws.tfvars"
echo ""
