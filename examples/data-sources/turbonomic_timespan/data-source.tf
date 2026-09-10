data "turbonomic_timespan" "all" {}

data "turbonomic_timespan" "business_hours" {
  display_name = "Business Hours"
}
