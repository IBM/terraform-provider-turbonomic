data "turbonomic_aws_eks_cluster" "example" {
  name          = "<cluster_name>"
  default_state = "RUNNING"
}
