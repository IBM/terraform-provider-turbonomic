data "turbonomic_target" "all" {}

output "target_count" {
  value = length(data.turbonomic_target.all.targets)
}
