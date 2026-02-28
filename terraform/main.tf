# Terraform 配置 - AWS Provider
terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "environment" {
  description = "Environment (dev/staging/prod)"
  type        = string
  default     = "dev"
}

variable "project_name" {
  description = "Project name"
  type        = string
  default     = "fulfillment-service"
}

# DynamoDB 表：存储订单数据
# NoSQL 数据库，只需定义主键，其他字段灵活添加
resource "aws_dynamodb_table" "orders" {
  name         = "${var.project_name}-orders-${var.environment}"
  billing_mode = "PAY_PER_REQUEST"  # 按请求付费，无需预留容量
  hash_key     = "OrderID"          # 分区键（主键）

  attribute {
    name = "OrderID"
    type = "S"  # String 类型
  }

  tags = {
    Name        = "${var.project_name}-orders"
    Environment = var.environment
  }
}

# 死信队列 (DLQ)：存储处理失败的消息
# 当消息处理失败 3 次后，自动移到这里，避免无限重试
resource "aws_sqs_queue" "orders_dlq" {
  name                      = "${var.project_name}-orders-dlq-${var.environment}"
  message_retention_seconds = 1209600  # 保留 14 天，方便排查问题

  tags = {
    Name        = "${var.project_name}-orders-dlq"
    Environment = var.environment
  }
}

# 主 SQS 队列：接收订单消息
# Producer 发消息到这里 → 触发 Lambda 处理
resource "aws_sqs_queue" "orders" {
  name                       = "${var.project_name}-orders-${var.environment}"
  visibility_timeout_seconds = 300     # 消息被读取后隐藏 5 分钟（Lambda 超时时间 + 缓冲）
  message_retention_seconds  = 345600  # 消息保留 4 天

  # DLQ 配置：处理失败 3 次后移到死信队列
  # ⚠️ 这是 SQS 的功能，不是代码里配置的！
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.orders_dlq.arn  # 死信队列 ARN
    maxReceiveCount     = 3  # 最多重试 3 次
  })

  tags = {
    Name        = "${var.project_name}-orders"
    Environment = var.environment
  }
}

# IAM 角色：Lambda 的"身份证"
# Lambda 需要这个角色才能访问其他 AWS 服务
resource "aws_iam_role" "lambda_execution" {
  name = "${var.project_name}-lambda-execution-${var.environment}"

  # 信任策略：允许 Lambda 服务扮演这个角色
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"  # 只有 Lambda 可以使用这个角色
        }
      }
    ]
  })
}

# IAM 权限策略：Lambda 能做什么
# 最小权限原则：只给需要的权限
resource "aws_iam_role_policy" "lambda_policy" {
  name = "${var.project_name}-lambda-policy"
  role = aws_iam_role.lambda_execution.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        # 权限 1: 写日志到 CloudWatch
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = "arn:aws:logs:*:*:*"
      },
      {
        # 权限 2: 从 SQS 读取和删除消息
        # AWS 会自动调用这些 API，不是你的代码调用
        Effect = "Allow"
        Action = [
          "sqs:ReceiveMessage",  # 读取消息
          "sqs:DeleteMessage",   # 处理成功后删除
          "sqs:GetQueueAttributes"
        ]
        Resource = aws_sqs_queue.orders.arn
      },
      {
        # 权限 3: 操作 DynamoDB
        Effect = "Allow"
        Action = [
          "dynamodb:PutItem",    # 你的代码调用
          "dynamodb:GetItem",    # 你的代码调用
          "dynamodb:UpdateItem"  # 你的代码调用
        ]
        Resource = aws_dynamodb_table.orders.arn
      }
    ]
  })
}

# Lambda 函数：订单处理器
# 上传你编译好的 Go 代码
resource "aws_lambda_function" "order_processor" {
  filename         = "${path.module}/../lambda.zip"  # build.bat 生成的文件
  function_name    = "${var.project_name}-order-processor-${var.environment}"
  role            = aws_iam_role.lambda_execution.arn  # 使用上面创建的角色
  handler         = "bootstrap"  # Go 自定义 runtime 的固定入口名
  source_code_hash = filebase64sha256("${path.module}/../lambda.zip")  # 用于检测代码变更
  runtime         = "provided.al2023"  # Amazon Linux 2023
  timeout         = 60  # 最长运行 60 秒

  # 环境变量：传递配置给代码
  environment {
    variables = {
      TABLE_NAME = aws_dynamodb_table.orders.name  # 你的代码用 os.Getenv("TABLE_NAME") 读取
    }
  }

  tags = {
    Name        = "${var.project_name}-order-processor"
    Environment = var.environment
  }
}

# Event Source Mapping：连接 SQS 和 Lambda
# 告诉 AWS："当 SQS 有消息时，自动调用 Lambda"
resource "aws_lambda_event_source_mapping" "sqs_trigger" {
  event_source_arn = aws_sqs_queue.orders.arn  # SQS 队列
  function_name    = aws_lambda_function.order_processor.arn  # Lambda 函数
  batch_size       = 10  # 一次最多传 10 条消息给 HandleSQSEvent
  enabled          = true
}

# 输出值：部署完成后显示这些信息
output "sqs_queue_url" {
  description = "SQS Queue URL for producer"
  value       = aws_sqs_queue.orders.url
}

output "dynamodb_table_name" {
  description = "DynamoDB Table Name"
  value       = aws_dynamodb_table.orders.name
}

output "lambda_function_name" {
  description = "Lambda Function Name"
  value       = aws_lambda_function.order_processor.function_name
}
