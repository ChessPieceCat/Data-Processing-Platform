resource "aws_instance" "app" {
  ami                    = "ami-04d0f56e9ce314a8e"
  instance_type          = "t4g.micro"
  subnet_id              = var.app_subnet_id
  vpc_security_group_ids = [aws_security_group.app.id]
  key_name               = "db1"
  iam_instance_profile   = aws_iam_instance_profile.ec2.name
  ebs_optimized          = true
  monitoring             = false

  root_block_device {
    volume_size = 16
    volume_type = "gp3"
    iops        = 3000
    throughput  = 125
  }

  tags = {
    Name = "data-processing-platform"
  }
}