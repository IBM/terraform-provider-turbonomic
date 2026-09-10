data "turbonomic_workflow" "all" {}

data "turbonomic_workflow" "webhooks" {
  type = "WEBHOOK"
}

data "turbonomic_workflow" "resize_webhook" {
  display_name = "Resize Approval Webhook"
}
