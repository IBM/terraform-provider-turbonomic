# List all schedules
data "turbonomic_schedule" "all" {}

# Look up a specific schedule by name
data "turbonomic_schedule" "maintenance" {
  display_name = "Maintenance Window"
}

# Reference the UUID in a settings policy
resource "turbonomic_settings_policy" "with_schedule" {
  display_name  = "Scheduled VM Resize Policy"
  schedule_uuid = data.turbonomic_schedule.maintenance.schedules[0].uuid

  settings_managers = [
    {
      uuid = "resizeVmSettingsManager"
      settings = [
        {
          uuid  = "resize"
          value = "RECOMMEND"
        }
      ]
    }
  ]
}
