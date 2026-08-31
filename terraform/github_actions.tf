resource "aws_iam_openid_connect_provider" "github" {
  url = "https://token.actions.githubusercontent.com"

  client_id_list = [
    "sts.amazonaws.com"
  ]
}

resource "aws_iam_role" "github_actions" {
  name = "data-processing-platform-github-actions"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Federated = aws_iam_openid_connect_provider.github.arn
        }
        Action = "sts:AssumeRoleWithWebIdentity"

        Condition = {
          StringEquals = {
            "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com"
            "token.actions.githubusercontent.com:sub" = "repo:ChessPieceCat@162753184/Data-Processing-Platform@1343320648:ref:refs/heads/main"
          }
        }
      }
    ]
  })
}

resource "aws_iam_role_policy" "github_actions_ecr" {
  name = "ecr-push"
  role = aws_iam_role.github_actions.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ecr:GetAuthorizationToken"
        ]
        Resource = "*"
      },
      {
        Effect = "Allow"
        Action = [
          "ecr:BatchCheckLayerAvailability",
          "ecr:CompleteLayerUpload",
          "ecr:InitiateLayerUpload",
          "ecr:PutImage",
          "ecr:UploadLayerPart"
        ]
        Resource = aws_ecr_repository.app.arn
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "github_actions_read_only" {
  role       = aws_iam_role.github_actions.name
  policy_arn = "arn:aws:iam::aws:policy/ReadOnlyAccess"
}

resource "aws_iam_role_policy" "github_actions_state" {
  name = "terraform-state"
  role = aws_iam_role.github_actions.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "ListStateBucket"
        Effect = "Allow"
        Action = [
          "s3:ListBucket"
        ]
        Resource = "arn:aws:s3:::data-processing-platform-terraform-state-410126553529"
        Condition = {
          StringEquals = {
            "s3:prefix" = "data-processing-platform/terraform.tfstate"
          }
        }
      },
      {
        Sid    = "ManageState"
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject"
        ]
        Resource = "arn:aws:s3:::data-processing-platform-terraform-state-410126553529/data-processing-platform/terraform.tfstate"
      },
      {
        Sid    = "ManageStateLock"
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject",
          "s3:DeleteObject"
        ]
        Resource = "arn:aws:s3:::data-processing-platform-terraform-state-410126553529/data-processing-platform/terraform.tfstate.tflock"
      }
    ]
  })
}

resource "aws_iam_role" "github_terraform_plan" {
  name = "data-processing-platform-github-terraform-plan"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Federated = aws_iam_openid_connect_provider.github.arn
        }
        Action = "sts:AssumeRoleWithWebIdentity"

        Condition = {
          StringEquals = {
            "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com"
            "token.actions.githubusercontent.com:sub" = "repo:ChessPieceCat@162753184/Data-Processing-Platform@1343320648:pull_request"
          }
        }
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "github_terraform_read_only" {
  role       = aws_iam_role.github_terraform_plan.name
  policy_arn = "arn:aws:iam::aws:policy/ReadOnlyAccess"
}

resource "aws_iam_role_policy" "github_terraform_state" {
  name = "terraform-state"
  role = aws_iam_role.github_terraform_plan.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "ListStateBucket"
        Effect = "Allow"
        Action = [
          "s3:ListBucket"
        ]
        Resource = "arn:aws:s3:::data-processing-platform-terraform-state-410126553529"
        Condition = {
          StringEquals = {
            "s3:prefix" = "data-processing-platform/terraform.tfstate"
          }
        }
      },
      {
        Sid    = "ManageState"
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject"
        ]
        Resource = "arn:aws:s3:::data-processing-platform-terraform-state-410126553529/data-processing-platform/terraform.tfstate"
      },
      {
        Sid    = "ManageStateLock"
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject",
          "s3:DeleteObject"
        ]
        Resource = "arn:aws:s3:::data-processing-platform-terraform-state-410126553529/data-processing-platform/terraform.tfstate.tflock"
      }
    ]
  })
}