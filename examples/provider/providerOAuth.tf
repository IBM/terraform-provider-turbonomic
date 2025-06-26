terraform {
  required_providers {
    turbonomic = {
      source  = "IBM/turbonomic"
      version = "1.2.0"
    }
  }
}

provider "turbonomic" {
  hostname      = var.hostname
  client_id     = var.client_id
  client_secret = var.client_secret
  role          = var.role
  skipverify    = var.skipverify
}
