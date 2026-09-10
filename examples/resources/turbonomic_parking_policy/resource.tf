data "turbonomic_timespan" "office_hours" {
  display_name = "Office Hours"
}

resource "turbonomic_parking_policy" "dev_vms" {
  display_name        = "Park Dev VMs Outside Office Hours"
  level               = "ACCOUNT"
  restrict_unparkable = false

  attach_schedule {
    timespan_schedule_uuid = data.turbonomic_timespan.office_hours.timespans[0].uuid
  }

  criteria_list {
    filter_type = "vmsByName"
    exp_type    = "RXEQ"
    exp_val     = "^dev-"
  }
}

resource "turbonomic_parking_policy" "global_weekend" {
  display_name        = "Global Weekend Parking"
  level               = "GLOBAL"
  priority            = 10
  restrict_unparkable = true

  criteria_list {
    filter_type = "vmsByTag"
    exp_type    = "EQ"
    exp_val     = "env:nonprod"
  }
}
