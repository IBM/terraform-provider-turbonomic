#result : {turbonomic_optimized_by = "turbonomic-terraform-provider"}

output "turbonomic_tag" {
  value = provider::turbonomic::get_tag()
}

