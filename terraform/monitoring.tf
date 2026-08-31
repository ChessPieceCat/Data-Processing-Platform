resource "aws_cloudwatch_log_group" "server" {
  name = "/data-processing-platform/server"
}

resource "aws_cloudwatch_log_group" "worker" {
  name = "/data-processing-platform/worker"
}

resource "aws_cloudwatch_metric_alarm" "high_cpu" {
  alarm_name          = "DataProcessingPlatform-HighCPU"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  datapoints_to_alarm = 2
  metric_name         = "CPUUtilization"
  namespace           = "AWS/EC2"
  period              = 300
  statistic           = "Average"
  threshold           = 80
  treat_missing_data  = "missing"

  dimensions = {
    InstanceId = aws_instance.app.id
  }
}

resource "aws_cloudwatch_metric_alarm" "high_disk" {
  alarm_name          = "DataProcessingPlatform-HighDisk"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  datapoints_to_alarm = 2
  metric_name         = "disk_used_percent"
  namespace           = "DataProcessingPlatform"
  period              = 300
  statistic           = "Average"
  threshold           = 90
  treat_missing_data  = "missing"

  dimensions = {
    path   = "/"
    host   = "ip-172-31-45-194"
    device = "nvme0n1p1"
    fstype = "ext4"
  }
}

resource "aws_cloudwatch_metric_alarm" "high_memory" {
  alarm_name          = "DataProcessingPlatform-HighMemory"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  datapoints_to_alarm = 2
  metric_name         = "mem_used_percent"
  namespace           = "DataProcessingPlatform"
  period              = 300
  statistic           = "Average"
  threshold           = 85
  treat_missing_data  = "missing"

  dimensions = {
    host = "ip-172-31-45-194"
  }
}

resource "aws_cloudwatch_metric_alarm" "status_check" {
  alarm_name          = "DataProcessingPlatform-StatusCheck"
  alarm_description   = "Alarm on instance i-039ced26aad174d78: Triggered when StatusCheckFailed >= 0.99 for 1 consecutive 5-minute periods."
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = 1
  metric_name         = "StatusCheckFailed"
  namespace           = "AWS/EC2"
  period              = 300
  statistic           = "Average"
  threshold           = 0.99
  treat_missing_data  = "missing"

  dimensions = {
    InstanceId = aws_instance.app.id
  }
}

resource "aws_cloudwatch_dashboard" "main" {
  dashboard_name = "Data-Processing-Platform"

  dashboard_body = jsonencode({
    widgets = [
      {
        type   = "alarm"
        x      = 0
        y      = 0
        width  = 15
        height = 2
        properties = {
          title = ""
          alarms = [
            aws_cloudwatch_metric_alarm.high_cpu.arn,
            aws_cloudwatch_metric_alarm.high_disk.arn,
            aws_cloudwatch_metric_alarm.high_memory.arn,
            aws_cloudwatch_metric_alarm.status_check.arn
          ]
        }
      }
    ]
  })
}