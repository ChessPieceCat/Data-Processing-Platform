output "aws_region" {
  description = "AWS region used by the deployment."
  value       = var.aws_region
}

output "vpc_id" {
  description = "VPC used by the deployment."
  value       = data.aws_vpc.default.id
}

output "app_subnet_id" {
  description = "Subnet used by the application EC2 instance."
  value       = data.aws_subnet.app.id
}

output "instance_id" {
  description = "EC2 instance ID."
  value       = aws_instance.app.id
}

output "public_ip" {
  description = "Public IPv4 address assigned to the EC2 instance."
  value       = aws_instance.app.public_ip
}

output "s3_bucket_name" {
  description = "S3 bucket used for job artifacts."
  value       = aws_s3_bucket.jobs.id
}

output "rds_endpoint" {
  description = "RDS endpoint."
  value       = aws_db_instance.main.address
}

output "rds_port" {
  description = "RDS PostgreSQL port."
  value       = aws_db_instance.main.port
}

output "app_security_group_id" {
  description = "Application EC2 security group ID."
  value       = aws_security_group.app.id
}

output "db_security_group_id" {
  description = "RDS security group ID."
  value       = aws_security_group.db.id
}