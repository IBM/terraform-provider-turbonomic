terraform {
  required_providers {
    turbonomic = {
      source  = "github.ibm.com/turbonomic/terraform-provider-turbonomic"
      version = "1.0.0"
    }
  }
}

provider "turbonomic" {
  username   = var.username
  password   = var.password
  hostname   = var.hostname
  skipverify = var.skipverify
}
