## 1.2.0 (Beta Release)

FEATURES:

- **New Data Resource:** `turbonomic_aws_db_instance`
- **New Data Resource:** `turbonomic_azurerm_managed_disk`
- **New Data Resource:** `turbonomic_aws_ebs_volume`
- **New Data Resource:** `turbonomic_google_compute_disk`
- Added `ApiInfo` to send the basic metadata info to the go-client.
- **New Function:** `provider::turbonomic::get_tag()`

NOTES:

- Provider now requires `Terraform v1.8.5`
- Update provier to use `github.com/IBM/turbonomic-go-client-v1.2.0`

## 1.1.0 (Beta Release)

FEATURES:

- Added `default_size` field in data source block, which replaces the need for checking `new_instance_type` for null
- Added oAuth 2.0 support for authenticating to Turbonomic's API
  - See [Creating and authenticating an OAuth 2.0 client](https://www.ibm.com/docs/en/tarm/8.15.0?topic=cookbook-authenticating-oauth-20-clients-api#cookbook_administration_oauth_authentication__title__4) for details

NOTES:

- **provider:** Update provider to use Go `1.23.7`, `github.com/IBM/turbonomic-go-client-v1.1.0` and `golang.org/x/net-v0.36.0`

## 1.0.1 (Beta Release)

BUG FIXES:

- **data-source/turbonomic_cloud_entity_recommendation:** Fixed issue with `turbonomic_cloud_data_source` data source where an error is throw when specifying a entity that does not exist


## 1.0.0 (Beta Release)

FEATURES:

- **New Data Resource:** `turbonomic_cloud_data_source`
