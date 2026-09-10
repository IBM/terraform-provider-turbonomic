resource "turbonomic_target" "aws" {
  type     = "AWS"
  category = "Public Cloud"

  input_fields = {
    address     = "AWS Production"
    iamRole     = "false"
    iamRoleArn  = ""
    accountType = "Standard"
  }

  input_fields_wo = {
    username = var.aws_access_key_id
    password = var.aws_secret_access_key
  }

  input_fields_wo_version = 1
}
