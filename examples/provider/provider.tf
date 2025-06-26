terraform {
  required_providers {
    turbonomic = {
      source  = "IBM/turbonomic"
      version = "1.2.0"
    }
  }
}

provider "turbonomic" {
  hostname   = var.hostname
  username   = var.username
  password   = var.password
  skipverify = var.skipverify
}
