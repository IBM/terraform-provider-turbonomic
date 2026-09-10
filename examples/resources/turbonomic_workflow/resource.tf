resource "turbonomic_workflow" "resize_webhook" {
  display_name       = "Resize Approval Webhook"
  type               = "WEBHOOK"
  action_type        = "RESIZE"
  action_phase       = "PRE"
  entity_type        = "VirtualMachine"
  time_limit_seconds = 300

  webhook_url         = "https://webhook.example.com/approve"
  webhook_method      = "POST"
  webhook_auth_method = "BASIC"
  webhook_username    = "turbo-webhook"
  webhook_password    = var.webhook_password
  webhook_template = jsonencode({
    action  = "{{action.actionType}}"
    entity  = "{{target.displayName}}"
    risk    = "{{action.risk}}"
    savings = "{{action.savings}}"
  })
}

resource "turbonomic_workflow" "notification_webhook" {
  display_name = "Post-Resize Notification"
  type         = "WEBHOOK"
  action_type  = "RESIZE"
  action_phase = "POST"

  webhook_url    = "https://hooks.slack.com/services/example"
  webhook_method = "POST"
  webhook_template = jsonencode({
    text = "Turbonomic completed resize for {{target.displayName}}"
  })
}
