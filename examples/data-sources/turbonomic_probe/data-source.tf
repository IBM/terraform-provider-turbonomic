data "turbonomic_probe" "all" {}

data "turbonomic_probe" "kubernetes" {
  type = "Kubernetes"
}

data "turbonomic_probe" "cloud_native" {
  category = "Cloud Native"
}
