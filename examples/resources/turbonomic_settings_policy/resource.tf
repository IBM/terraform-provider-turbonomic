
# Automation settings policy for VMs scoped to a specific group
resource "turbonomic_settings_policy" "vm_automation" {
  name        = "prod-vm-automation-policy"
  entity_type = "VirtualMachine"
  disabled    = false
  scope_uuids = [turbonomic_group.prod_vms.uuid]
  note        = "Managed by Terraform"

  settings_managers {
    category = "automationmanager"

    settings {
      uuid  = "moveVirtualMachine"
      value = "AUTOMATIC"
    }

    settings {
      uuid  = "resizeVirtualMachine"
      value = "RECOMMEND"
    }
  }

  settings_managers {
    category = "marketsettingsmanager"

    settings {
      uuid  = "resizeTargetUtilizationVcpu"
      value = "70.0"
    }

    settings {
      uuid  = "resizeTargetUtilizationVmem"
      value = "80.0"
    }
  }
}

# Global settings policy (no scope = applies to all VMs)
resource "turbonomic_settings_policy" "vm_global_defaults" {
  name        = "global-vm-defaults"
  entity_type = "VirtualMachine"

  settings_managers {
    category = "automationmanager"

    settings {
      uuid  = "moveVirtualMachine"
      value = "RECOMMEND"
    }
  }
}
