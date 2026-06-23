provider "aws" {
  region = "us-east-1"
}

# Query existing AWS instance (if it exists)
data "aws_instance" "existing" {
  instance_id = "i-0021d61fa77f000d0" # Replace with your instance ID or use filters
}

# Query Turbonomic for recommendations
data "turbonomic_aws_instance" "example" {
  entity_name = "my-ec2-instance"
  vendor_id   = "i-0021d61fa77f000d0"
  # default_instance_type is intentionally omitted to enable fallback pattern
}

# Create or update the instance with fallback logic
# The coalesce() function creates a priority chain:
# 1. Use Turbonomic recommendation if available
# 2. Use current AWS instance type if Turbonomic is unavailable
# 3. Use default "t2.nano" for new VMs
resource "aws_instance" "terraform-demo-ec2" {
  ami = "ami-079db87dc4c10ac91"
  instance_type = coalesce(
    data.turbonomic_aws_instance.example.new_instance_type, # Turbonomic recommendation
    try(data.aws_instance.existing.instance_type, null),    # Current AWS instance type
    "t2.nano"                                               # Default for new VMs
  )

  tags = merge(
    {
      Name = "my-ec2-instance"
    },
    provider::turbonomic::get_tag() # Tag the resource as optimized by Turbonomic provider
  )
}

# Output the selected instance type for verification
output "selected_instance_type" {
  description = "The instance type selected by the fallback pattern"
  value       = aws_instance.terraform-demo-ec2.instance_type
}

output "turbonomic_recommendation" {
  description = "The instance type recommended by Turbonomic (if available)"
  value       = data.turbonomic_aws_instance.example.new_instance_type
}

output "current_instance_type" {
  description = "The current AWS instance type (if exists)"
  value       = try(data.aws_instance.existing.instance_type, "N/A - New instance")
}
