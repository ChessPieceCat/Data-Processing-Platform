variable "aws_region" {
  description = "AWS region for the deployment."
  type        = string
  default     = "us-east-2"
}

variable "app_subnet_id" {
  description = "Subnet used by the application EC2 instance."
  type        = string
  default     = "subnet-00f473a9177e9fa0e"
}

variable "ssh_cidr" {
  description = "CIDR allowed to SSH to the application EC2 instance."
  type        = string
}

variable "account_id" {
  description = "AWS account ID."
  type        = string
  default     = "410126553529"
}

variable "db_secret_arn" {
  description = "RDS master credentials secret ARN."
  type        = string
  sensitive   = true
}