resource "turbonomic_schedule" "one_time" {
  display_name = "One-Time Maintenance"
  start_date   = "2025-12-01T00:00"
  start_time   = "2000-01-01T22:00"
  end_time     = "2000-01-02T06:00"
  time_zone    = "America/New_York"
}

resource "turbonomic_schedule" "weekly" {
  display_name = "Weekend Maintenance Window"
  start_date   = "2025-01-01T00:00"
  start_time   = "2000-01-01T22:00"
  end_time     = "2000-01-02T06:00"
  time_zone    = "UTC"

  recurrence = {
    type         = "WEEKLY"
    days_of_week = ["Sat", "Sun"]
  }
}

resource "turbonomic_schedule" "daily" {
  display_name = "Nightly Automation Window"
  start_date   = "2025-01-01T00:00"
  start_time   = "2000-01-01T02:00"
  end_time     = "2000-01-01T04:00"
  time_zone    = "UTC"

  recurrence = {
    type     = "DAILY"
    interval = 1
  }
}

resource "turbonomic_schedule" "monthly" {
  display_name = "Monthly Patch Window"
  start_date   = "2025-01-01T00:00"
  start_time   = "2000-01-01T22:00"
  end_time     = "2000-01-02T06:00"
  time_zone    = "UTC"

  recurrence = {
    type          = "MONTHLY"
    days_of_month = [1]
  }
}

# Reference the weekly schedule UUID in a settings policy
resource "turbonomic_settings_policy" "with_schedule" {
  name          = "weekend-vm-automation"
  entity_type   = "VirtualMachine"
  schedule_uuid = turbonomic_schedule.weekly.uuid

  settings_managers {
    category = "automationmanager"
    settings {
      uuid  = "moveVirtualMachine"
      value = "AUTOMATIC"
    }
  }
}
