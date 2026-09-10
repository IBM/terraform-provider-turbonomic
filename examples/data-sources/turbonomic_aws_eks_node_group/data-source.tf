data "turbonomic_aws_eks_node_group" "example" {
  cluster_name         = "<cluster_name>"
  node_group_name      = "<node_group_name>"
  default_desired_size = 3
}
