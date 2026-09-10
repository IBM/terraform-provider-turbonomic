---
page_title: "Managing Targets"
description: |-
  This guide covers the turbonomic_target resource and data source for managing Turbonomic targets (probes) with Terraform.
---

# Managing targets

Turbonomic targets are the external systems  - hypervisors, cloud accounts, monitoring tools, and so on  - that Turbonomic discovers and monitors to build its connected virtual environment model. This guide covers the `turbonomic_target` resource and `turbonomic_target` data source added to the Turbonomic Terraform Provider.

## turbonomic_target data source

Returns all Turbonomic targets (probes) registered with your Turbonomic instance, including their health state, audit metadata, and non-secret configuration fields.

### All targets

```terraform
data "turbonomic_target" "all" {}

output "target_count" {
  value = length(data.turbonomic_target.all.targets)
}
```

### Filter by probe type

```terraform
data "turbonomic_target" "aws" {
  type = "AWS"
}

output "aws_targets" {
  value = data.turbonomic_target.aws.targets
}
```

## turbonomic_target resource

Manages a Turbonomic target (probe). Turbonomic validates and discovers the target after it is created or updated. The computed health attributes reflect the result of the most recent discovery cycle.

-> **Note** Requires Terraform 1.10+ when using `input_fields_wo` with ephemeral values.

### vCenter (Hypervisor)

```terraform
resource "turbonomic_target" "vcenter" {
  type     = "vCenter"
  category = "Hypervisor"

  input_fields = {
    address  = var.vcenter_host
    username = var.vcenter_username
  }

  input_fields_wo = {
    password = var.vcenter_password
  }

  input_fields_wo_version = 1
}
```

### AWS  - access key authentication

```terraform
resource "turbonomic_target" "aws" {
  type     = "AWS"
  category = "Public Cloud"

  input_fields = {
    address     = "AWS Production"
    iamRole     = "false"
    iamRoleArn  = ""
    accountType = "Standard"
  }

  input_fields_wo = {
    username = var.aws_access_key_id
    password = var.aws_secret_access_key
  }

  input_fields_wo_version = 1
}
```

### GCP

```terraform
resource "turbonomic_target" "gcp" {
  type     = "GCP Service Account"
  category = "Public Cloud"

  input_fields = {
    name = "GCP Production"
  }

  input_fields_wo = {
    serviceAccountKey = file(var.gcp_credentials_file)
  }

  input_fields_wo_version = 1
}
```

### Azure Service Principal

```terraform
resource "turbonomic_target" "azure" {
  type     = "Azure Service Principal"
  category = "Public Cloud"

  input_fields = {
    name   = "Azure Production"
    tenant = var.azure_tenant_id
    client = var.azure_client_id
  }

  input_fields_wo = {
    key = var.azure_client_secret
  }

  input_fields_wo_version = 1
}
```

### HCP Terraform

```terraform
resource "turbonomic_target" "terraform_hcp" {
  type     = "Terraform"
  category = "Infrastructure as Code"

  input_fields = {
    displayName                = "HCP Terraform"
    resourceType               = "HCP"
    hcpOrganizationName        = var.hcp_org_name
    terraformHostname          = "app.terraform.io"
    validateServerCertificates = "true"
  }

  input_fields_wo = {
    hcpToken = var.hcp_token
  }

  input_fields_wo_version = 1
}
```

### ServiceNow

```terraform
resource "turbonomic_target" "servicenow" {
  type     = "ServiceNow"
  category = "IT Management"

  input_fields = {
    displayName   = "ServiceNow Production"
    targetAddress = var.servicenow_hostname
    username      = var.servicenow_username
  }

  input_fields_wo = {
    password = var.servicenow_password
  }

  input_fields_wo_version = 1
}
```

### Instana

```terraform
resource "turbonomic_target" "instana" {
  type     = "Instana"
  category = "Applications and Databases"

  input_fields = {
    address  = var.instana_hostname
    apiToken = var.instana_api_token
  }

  input_fields_wo_version = 1
}
```

## Secret rotation with Vault (ephemeral values)

Write-only credentials in `input_fields_wo` are never stored in Terraform state. To rotate a secret without exposing it in state, source it from an ephemeral data source (e.g. HashiCorp Vault) and increment `input_fields_wo_version` to trigger a re-apply.

```terraform
# Ephemeral data source - value never stored in state
ephemeral "vault_kv_secret_v2" "hcp_credentials" {
  mount = "kv"
  name  = "turbonomic/terraform/hcp"
}

resource "turbonomic_target" "terraform_hcp_vault" {
  type     = "Terraform"
  category = "Infrastructure as Code"

  input_fields = {
    displayName                = "HCP Terraform"
    resourceType               = "HCP"
    hcpOrganizationName        = var.hcp_org_name
    terraformHostname          = "app.terraform.io"
    validateServerCertificates = "true"
  }

  # Sensitive fields using ephemeral values (NEVER stored in state)
  input_fields_wo = {
    hcpToken = tostring(ephemeral.vault_kv_secret_v2.hcp_credentials.data["token"])
  }

  # Increment to trigger re-send of credentials on next apply
  input_fields_wo_version = 1
}
```

When a secret is rotated in Vault:

1. **Secret Updated in Vault**: Administrator rotates the token in Vault.
2. **Increment Version**: Update `input_fields_wo_version` in the Terraform configuration (e.g., from `1` to `2`).
3. **Terraform Detects Change**: On `terraform plan`, Terraform detects the version change.
4. **Fresh Secret Fetched**: The ephemeral data source fetches the new token from Vault.
5. **Manual Apply Required**: Run `terraform apply` to update the target.
6. **Target Updated**: Turbonomic target is updated with new credentials via the API.
7. **No State Pollution**: The new secret is sent to the API but never written to state.

## Probe types

The `type` attribute must exactly match a probe registered in your Turbonomic instance. Common values by category:

### Public cloud

| `type` value | Description |
|---|---|
| `AWS` | Amazon Web Services |
| `Azure Service Principal` | Azure Service Principal |
| `GCP Service Account` | GCP Service Account |
| `OCI Tenancy` | Oracle Cloud Infrastructure Tenancy |
| `SoftLayer` | IBM Cloud / SoftLayer |

### Virtualization & private cloud

| `type` value | Description |
|---|---|
| `vCenter` | VMware vCenter |
| `Hyper-V` | Microsoft Hyper-V |
| `Nutanix` | Nutanix Acropolis |
| `OpenStack` | OpenStack |
| `Red Hat Virtualization Manager` | Red Hat Virtualization (RHV) |

### Cloud native & containers

| `type` value | Description |
|---|---|
| `Kubernetes` | Kubernetes |
| `Istio` | Istio service mesh |

### Infrastructure as code

| `type` value | Description |
|---|---|
| `Terraform` | HashiCorp / IBM Terraform |
| `GitHub` | GitHub |
| `GitLab` | GitLab |
| `AzureDevOps` | Azure DevOps |

### APM & observability

| `type` value | Description |
|---|---|
| `AppDynamics` | Cisco AppDynamics |
| `Dynatrace` | Dynatrace |
| `Datadog` | Datadog |
| `NewRelic` | New Relic |
| `Instana` | IBM Instana |
| `Prometheus` | Prometheus |

### ITSM & integrations

| `type` value | Description |
|---|---|
| `ServiceNow` | ServiceNow |
| `Webhook` | Webhook |

See the [turbonomic_target resource](https://registry.terraform.io/providers/IBM/turbonomic/latest/docs/resources/target) documentation for the complete list of probe types.

## Finding input field names

Input field names are specific to each probe and are not validated by the provider. To discover the exact field names for a given probe type, query your Turbonomic instance:

```shell
curl -sk -b <session-cookie> \
  "https://<turbonomic-host>/api/v3/targets/specs" \
  | jq '.[] | select(.type == "<ProbeType>")'
```

Replace `<ProbeType>` with the exact `type` value (e.g. `vCenter`, `AWS`, `Terraform`).

## Import

Existing targets can be imported using their UUID:

```shell
terraform import turbonomic_target.example <uuid>
```

After import, `input_fields_wo` will be empty in state because write-only values are never persisted. Populate `input_fields_wo` and increment `input_fields_wo_version` in your configuration, then run `terraform apply` to re-send the credentials.

## Notes

- **Field names are probe-specific.** Each probe type exposes different `InputField` names. Use the target specs endpoint to discover the correct names for any probe type.
- **All field values are strings.** Even numeric or boolean probe fields (e.g. `port`, `iamRole`, `validateServerCertificates`) must be passed as string values.
- **Write-only credentials are never stored in state.** To rotate a secret, update `input_fields_wo` and increment `input_fields_wo_version`.
- **Health attributes are populated after discovery.** On initial creation the health fields will be empty until Turbonomic completes its first discovery cycle. Run `terraform refresh` or a subsequent `terraform apply` to pick up the updated health status.
- **Probe must be registered.** Creating a target for a probe type that is not registered in the Turbonomic instance will result in an API error. Verify the probe is running before applying.
