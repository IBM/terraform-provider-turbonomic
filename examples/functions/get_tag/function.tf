terraform {
  required_providers {
    turbonomic = {
      source  = "IBM/turbonomic"
      version = "2.0.0"
    }
  }
}

# Emit the tag object directly
output "turbonomic_tag" {
  value = provider::turbonomic::get_tag()
}
