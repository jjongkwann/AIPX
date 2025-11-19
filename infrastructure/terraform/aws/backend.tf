# Terraform Backend Configuration
# S3 버킷과 DynamoDB 테이블은 수동으로 먼저 생성해야 합니다.
#
# 생성 명령어:
# aws s3 mb s3://aipx-terraform-state --region ap-northeast-2
# aws s3api put-bucket-versioning --bucket aipx-terraform-state --versioning-configuration Status=Enabled
# aws dynamodb create-table --table-name aipx-terraform-locks --attribute-definitions AttributeName=LockID,AttributeType=S --key-schema AttributeName=LockID,KeyType=HASH --billing-mode PAY_PER_REQUEST --region ap-northeast-2

terraform {
  backend "s3" {
    bucket         = "aipx-terraform-state"
    key            = "aws/terraform.tfstate"
    region         = "ap-northeast-2"
    encrypt        = true
    dynamodb_table = "aipx-terraform-locks"
  }
}
