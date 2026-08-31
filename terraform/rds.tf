resource "aws_db_instance" "main" {
  identifier = "database-1"

  engine         = "postgres"
  engine_version = "18.3"
  instance_class = "db.t4g.micro"

  allocated_storage     = 20
  max_allocated_storage = 1000
  storage_type          = "gp2"

  db_name  = "data_platform"
  username = "dbadmin"

  port                         = 5432
  publicly_accessible          = false
  multi_az                     = false
  deletion_protection          = false
  backup_retention_period      = 1
  auto_minor_version_upgrade   = true
  copy_tags_to_snapshot        = true
  performance_insights_enabled = true

  db_subnet_group_name = "default-vpc-03cfbf5f7873399e7"

  vpc_security_group_ids = [
    aws_security_group.db.id
  ]

  parameter_group_name = "default.postgres18"

  storage_encrypted = true
  kms_key_id        = "arn:aws:kms:us-east-2:410126553529:key/52d1caca-475d-48c1-acc9-574100efea9b"

  backup_window      = "06:37-07:07"
  maintenance_window = "tue:03:10-tue:03:40"

}