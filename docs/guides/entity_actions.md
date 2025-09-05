---
page_title: "Entity actions data source"
subcategory: ""
description: |-
  This document explains the purpose of the entity actions data source and provides guidance on how to use it.
---

# Entity actions data source

The entity actions data source offers a flexible mechanism for retrieving actions from Turbonomic and accessing
their associated data via the API. This allows users to work with Turbonomic actions that may not be supported
by standard data sources, or to extract specific action-related details that other sources might not expose.

For example, if you're using the `turbonomic_aws_instance` data source but need to identify which commodity is
triggering a cloud scaling action, this data source can help bridge that gap.

Another use case involves scaling actions for Azure Cosmos DB containers. Currently, there isn't a dedicated
data source for this scenario, but the `turbonomic_entity_actions` data source can be leveraged to utilize
Turbonomic's recommendations effectively.

## Inputs

This data source requires two mandatory fields and supports three optional ones. The fields and their syntax are as follows:

```terraform
data "turbonomic_entity_actions" "testing" {
  entity_name      = "<entity_name>"
  entity_type      = "<entity_type>"
  action_types     = ["<action_types>"]
  environment_type = "<environment_type>"
  states           = ["<states>"]
}
```

### Required

- `entity_name` (String) case sensitive name of the entity
- `entity_type` (String) case insensitive type of the entity
    - Valid entity typed include `VirtualMachine`, `PhysicalMachine`, `Container`, `DatabaseServer` etc..  Refer to [Entity Reference Tables](https://www.ibm.com/docs/en/tarm/8.17.0?topic=reference-tables#appendix_reference__title__4)
    for a complete list

### Optional

You can use the following optional fields to filter which types of actions are returned by the data source:

- `action_types` (List of String) type of the action
    - Valid action types are `START`, `MOVE`, `SCALE`, `ALLOCATE`, `SUSPEND`, `PROVISION`, `RECONFIGURE`, `RESIZE`, `DELETE`, `RIGHT_SIZE`, `BUY_RI`
    - Multiple types can be specified
- `environment_type` (String) filter the actions by environment type
    - Valid environment types are `HYBRID`, `CLOU`, `ONPREM`, `UNKNOWN`
- `states` (List of String) list of action states to filter
    - Valid states include `ACCEPTED`, `REJECTED`, `SUCCEEDED`, `FAILED`, `READY` etc..  Refer to [Entity Reference Tables](https://www.ibm.com/docs/en/tarm/8.17.0?topic=reference-tables#appendix_reference__title__3)
    for a complete list
    - Multiple types can be specified

## Outputs

You can reference the schema for the outputs in [turbonomic_entity_actions](https://registry.terraform.io/providers/IBM/turbonomic/latest/docs/data-sources/entity_actions#schema)

## Example

The example below illustrates how to use the `turbonomic_entity_actions` data source to retrieve `READY` scaling actions for the `CLOUD`
Virtual Machine named `aws-scale-test`.

```terraform
data "turbonomic_entity_actions" "example" {
  entity_name      = "aws-scale-test"
  entity_type      = "VirtualMachine"
  action_types     = ["SCALE"]
  environment_type = "CLOUD"
  states           = ["READY"]
}
```

<details>
<summary>Example output:</summary>

```terraform
Changes to Outputs:
  + testing = {
      + action_types     = [
          + "SCALE",
        ]
      + actions          = [
          + {
              + action_id        = 639062647423794
              + action_impact_id = 639062647423794
              + action_mode      = "MANUAL"
              + action_state     = "READY"
              + action_type      = "SCALE"
              + compound_actions = [
                  + {
                      + action_mode      = "MANUAL"
                      + action_state     = "READY"
                      + action_type      = "SCALE"
                      + current_entity   = {
                          + class_name       = "ComputeTier"
                          + discovered_by    = {
                              + category            = ""
                              + display_name        = "Standard"
                              + is_probe_registered = false
                              + read_only           = false
                              + type                = "AWS Infrastructure"
                              + uuid                = "75878946158192"
                            }
                          + display_name     = "m5.2xlarge"
                          + environment_type = "CLOUD"
                          + state            = "ACTIVE"
                          + uuid             = "75878942128271"
                          + vendor_ids       = {
                              + Standard = "m5.2xlarge"
                            }
                        }
                      + current_value    = "75878942128271"
                      + details          = "Scale Virtual Machine aws-scale-test from m5.2xlarge to r6a.xlarge"
                      + display_name     = "MANUAL"
                      + new_entity       = {
                          + class_name       = "ComputeTier"
                          + discovered_by    = {
                              + category            = ""
                              + display_name        = "Standard"
                              + is_probe_registered = false
                              + read_only           = false
                              + type                = "AWS Infrastructure"
                              + uuid                = "75878946158192"
                            }
                          + display_name     = "r6a.xlarge"
                          + environment_type = "CLOUD"
                          + state            = "ACTIVE"
                          + uuid             = "75878942128303"
                          + vendor_ids       = {
                              + Standard = "r6a.xlarge"
                            }
                        }
                      + new_value        = "75878942128303"
                      + resize_attribute = ""
                      + risk             = {
                          + description        = ""
                          + importance         = 0
                          + reason_commodities = []
                          + severity           = ""
                          + sub_category       = ""
                        }
                      + target           = {
                          + aspects          = null
                          + class_name       = "VirtualMachine"
                          + discovered_by    = {
                              + category            = ""
                              + display_name        = "example.aws.amazon.com"
                              + is_probe_registered = false
                              + read_only           = false
                              + type                = "AWS"
                              + uuid                = "75878939124896"
                            }
                          + display_name     = "aws-scale-test"
                          + environment_type = "CLOUD"
                          + state            = "ACTIVE"
                          + tags             = {
                              + Name          = [
                                  + "aws-scale-test",
                                ]
                            }
                          + uuid             = "76112684294780"
                          + vendor_ids       = {
                              + "example.aws.amazon.com" = "i-0a0a000a00a00a000"
                            }
                        }
                      + value_units      = ""
                    },
                ]
              + create_time      = "2025-08-14T12:21:18Z"
              + current_entity   = {
                  + class_name       = "ComputeTier"
                  + discovered_by    = {
                      + category            = ""
                      + display_name        = "Standard"
                      + is_probe_registered = false
                      + read_only           = false
                      + type                = "AWS Infrastructure"
                      + uuid                = "75878946158192"
                    }
                  + display_name     = "m5.2xlarge"
                  + environment_type = "CLOUD"
                  + state            = "ACTIVE"
                  + uuid             = "75878942128271"
                  + vendor_ids       = {
                      + Standard = "m5.2xlarge"
                    }
                }
              + current_location = {
                  + class_name       = "Region"
                  + discovered_by    = {
                      + category            = "Public Cloud"
                      + display_name        = "Standard"
                      + is_probe_registered = false
                      + read_only           = false
                      + type                = "AWS Infrastructure"
                      + uuid                = "75878946158192"
                    }
                  + display_name     = "aws-EU (Frankfurt)"
                  + environment_type = "CLOUD"
                  + uuid             = "75878942121240"
                  + vendor_ids       = {
                      + Standard = "aws::eu-central-1::DC::eu-central-1"
                    }
                }
              + current_value    = "75878942128271"
              + details          = "Scale Virtual Machine aws-scale-test from m5.2xlarge to r6a.xlarge in 999999999999"
              + display_name     = "MANUAL"
              + importance       = 0
              + market_id        = 777777
              + new_entity       = {
                  + class_name       = "ComputeTier"
                  + discovered_by    = {
                      + category            = ""
                      + display_name        = "Standard"
                      + is_probe_registered = false
                      + read_only           = false
                      + type                = "AWS Infrastructure"
                      + uuid                = "75878946158192"
                    }
                  + display_name     = "r6a.xlarge"
                  + environment_type = "CLOUD"
                  + state            = "ACTIVE"
                  + uuid             = "75878942128303"
                  + vendor_ids       = {
                      + Standard = "r6a.xlarge"
                    }
                }
              + new_location     = {
                  + class_name       = "Region"
                  + discovered_by    = {
                      + category            = "Public Cloud"
                      + display_name        = "Standard"
                      + is_probe_registered = false
                      + read_only           = false
                      + type                = "AWS Infrastructure"
                      + uuid                = "75878946158192"
                    }
                  + display_name     = "aws-EU (Frankfurt)"
                  + environment_type = "CLOUD"
                  + uuid             = "75878942121240"
                  + vendor_ids       = {
                      + Standard = "aws::eu-central-1::DC::eu-central-1"
                    }
                }
              + new_value        = "75878942128303"
              + resize_attribute = ""
              + risk             = {
                  + description        = "Underutilized VCPU, Net Throughput"
                  + importance         = 0
                  + reason_commodities = [
                      + "NetThroughput",
                      + "VCPU",
                    ]
                  + severity           = "MINOR"
                  + sub_category       = "Efficiency Improvement"
                }
              + source           = "MARKET"
              + stats            = [
                  + {
                      + filters = [
                          + {
                              + display_name = ""
                              + type         = "savingsType"
                              + value        = "savings"
                            },
                        ]
                      + name    = "costPrice"
                      + units   = "$/h"
                      + value   = 0.1859517
                    },
                ]
              + target           = {
                  + aspects          = jsonencode(
                        {
                          + cloudAspect = {
                              + businessAccount = {
                                  + className   = "BusinessAccount"
                                  + displayName = "999999999999"
                                  + uuid        = "75878942104373"
                                }
                              + resourceId      = "arn:aws:ec2:eu-central-1:999999999999:instance/i-0a0a000a00a00a000"
                              + type            = "CloudAspectApiDTO"
                            }
                        }
                    )
                  + class_name       = "VirtualMachine"
                  + discovered_by    = {
                      + category            = ""
                      + display_name        = "example.aws.amazon.com"
                      + is_probe_registered = false
                      + read_only           = false
                      + type                = "AWS"
                      + uuid                = "75878939124896"
                    }
                  + display_name     = "aws-scale-test"
                  + environment_type = "CLOUD"
                  + state            = "ACTIVE"
                  + tags             = {
                      + Name          = [
                          + "aws-scale-test",
                        ]
                    }
                  + uuid             = "76112684294780"
                  + vendor_ids       = {
                      + "example.aws.amazon.com" = "i-0a0a000a00a00a000"
                    }
                }
              + template         = {
                  + class_name   = "ComputeTier"
                  + discovered   = false
                  + display_name = "r6a.xlarge"
                  + enable_match = false
                  + uuid         = "75878942128303"
                }
              + uuid             = "639062647423794"
              + value_units      = ""
            },
        ]
      + entity_name      = "aws-scale-test"
      + entity_type      = "VirtualMachine"
      + entity_uuid      = "76112684294780"
      + environment_type = "CLOUD"
      + states           = [
          + "READY",
        ]
    }
```

</details>


-> **NOTE:** Keep in mind that the actions attribute contains a list of actions. Depending on your query, you might receive multiple
actions in response. To improve the relevance of your results, we recommend using the input attributes to narrow your
search as much as possible.

## Use case: Rezising a Cosmos DB container's document collection

Although Turbonomic data sources currently don't support direct scaling of Cosmos DB document collections through
Turbonomic actions, you can still leverage the `turbonomic_entity_actions` data source to benefit from Turbonomic's
optimization recommendations.

Suppose you have a basic Cosmos DB setup defined using Terraform, similar to the example below:

```terraform
resource "azurerm_cosmosdb_sql_database" "example" {
  name                = "example-database"
  resource_group_name = "example-resource-group"
  account_name        = "example-cosmos-account"
}

resource "azurerm_cosmosdb_sql_container" "example" {
  name                = "example-doc-collection"
  resource_group_name = "example-resource-group"
  account_name        = "example-cosmos-account"
  database_name       = "example-cosmos-db"
  partition_key_paths = "/partitionKey"

  throughput = 400

  indexing_policy {
    indexing_mode = "consistent"

    included_path {
      path = "/*"
    }

    excluded_path {
      path = "/\"_etag\"/?"
    }
  }
}
```

As you can see, throughput is currently hard-coded. We can update this configuration to integrate the
`turbonomic_entity_actions` data source, allowing Turbonomic to serve as the authoritative source for throughput values.

First, we add the data source to our code:

```terraform
data "turbonomic_entity_actions" "example" {
  entity_name      = "example-doc-collection"
  entity_type      = "DocumentCollection"
  action_types     = ["SCALE"]
  environment_type = "CLOUD"
  states           = ["READY"]
}
```

The output from the data source is similar to the following:

<details>
<summary>Example output:</summary>

```terraform
Changes to Outputs:
  + testing = {
      + action_types     = [
          + "SCALE",
        ]
      + actions          = [
          + {
              + action_id        = 638828907944856
              + action_impact_id = 638828907944856
              + action_mode      = "MANUAL"
              + action_state     = "READY"
              + action_type      = "SCALE"
              + compound_actions = []
              + create_time      = "2025-08-18T08:01:53Z"
              + current_entity   = {
                  + class_name       = "DatabaseTier"
                  + discovered_by    = {
                      + category            = ""
                      + display_name        = "Example Azure Subscription"
                      + is_probe_registered = false
                      + read_only           = false
                      + type                = "Azure Subscription"
                      + uuid                = "75878940124304"
                    }
                  + display_name     = "cosmos"
                  + environment_type = "CLOUD"
                  + state            = "ACTIVE"
                  + uuid             = "75878941065234"
                  + vendor_ids       = {
                      + "Example Azure Subscription" = "azure::DBPROFILE::cosmos"
                    }
                }
              + current_location = {
                  + class_name       = ""
                  + discovered_by    = {
                      + category            = ""
                      + display_name        = ""
                      + is_probe_registered = false
                      + read_only           = false
                      + type                = ""
                      + uuid                = ""
                    }
                  + display_name     = ""
                  + environment_type = ""
                  + uuid             = ""
                  + vendor_ids       = null
                }
              + current_value    = "75878941065234"
              + details          = "Scale down RU for Document Collection example-doc-collection on cosmos from 1,000 to 400 in Example Azure Account"
              + display_name     = "MANUAL"
              + importance       = 0
              + market_id        = 777777
              + new_entity       = {
                  + class_name       = "DatabaseTier"
                  + discovered_by    = {
                      + category            = ""
                      + display_name        = "Example Azure Subscription"
                      + is_probe_registered = false
                      + read_only           = false
                      + type                = "Azure Subscription"
                      + uuid                = "75878940124304"
                    }
                  + display_name     = "cosmos"
                  + environment_type = "CLOUD"
                  + state            = "ACTIVE"
                  + uuid             = "75878941065234"
                  + vendor_ids       = {
                      + "Example Azure Subscription" = "azure::DBPROFILE::cosmos"
                    }
                }
              + new_location     = {
                  + class_name       = ""
                  + discovered_by    = {
                      + category            = ""
                      + display_name        = ""
                      + is_probe_registered = false
                      + read_only           = false
                      + type                = ""
                      + uuid                = ""
                    }
                  + display_name     = ""
                  + environment_type = ""
                  + uuid             = ""
                  + vendor_ids       = null
                }
              + new_value        = "75878941065234"
              + resize_attribute = ""
              + risk             = {
                  + description        = "Underutilized RU"
                  + importance         = 0
                  + reason_commodities = [
                      + "RU",
                    ]
                  + severity           = "MINOR"
                  + sub_category       = "Efficiency Improvement"
                }
              + source           = "MARKET"
              + stats            = [
                  + {
                      + filters = [
                          + {
                              + display_name = ""
                              + type         = "savingsType"
                              + value        = "savings"
                            },
                        ]
                      + name    = "costPrice"
                      + units   = "$/h"
                      + value   = 0.06
                    },
                ]
              + target           = {
                  + aspects          = jsonencode(
                        {
                          + cloudAspect = {
                              + businessAccount = {
                                  + className   = "BusinessAccount"
                                  + displayName = "Example Azure Account"
                                  + uuid        = "75878941495325"
                                }
                              + resourceGroup   = {
                                  + className   = "ResourceGroup"
                                  + displayName = "example-resource-group"
                                  + uuid        = "287223045491413"
                                }
                              + resourceId      = "/subscriptions/a000a0aa-0aaa-00aa-0000-aa000000a000/resourceGroups/example-resource-group/providers/Microsoft.DocumentDB/databaseAccounts/example-cosmos-account/sqlDatabases/example-cosmos-db/containers/example-doc-collection"
                              + type            = "CloudAspectApiDTO"
                            }
                        }
                    )
                  + class_name       = "DocumentCollection"
                  + discovered_by    = {
                      + category            = ""
                      + display_name        = "Example Azure Account"
                      + is_probe_registered = false
                      + read_only           = false
                      + type                = "Azure Subscription"
                      + uuid                = "75878940124306"
                    }
                  + display_name     = "example-doc-collection"
                  + environment_type = "CLOUD"
                  + state            = "ACTIVE"
                  + tags             = {
                      + defaultExperience = [
                          + "Core (SQL)",
                        ]
                      + key               = [
                          + "test",
                        ]
                    }
                  + uuid             = "75878942858920"
                  + vendor_ids       = {
                      + "Example Azure Account" = "azure::DOCUMENT_COLLECTION::AAA00A00-A000-00A0-AA00-0000A00A0A0A"
                    }
                }
              + template         = {
                  + class_name   = "DatabaseTier"
                  + discovered   = false
                  + display_name = "cosmos"
                  + enable_match = false
                  + uuid         = "75878941065234"
                }
              + uuid             = "638828907944856"
              + value_units      = ""
            },
        ]
      + entity_name      = "example-doc-collection"
      + entity_type      = "DocumentCollection"
      + entity_uuid      = "75878942858920"
      + environment_type = "CLOUD"
      + states           = [
          + "READY",
        ]
    }
```

</details>

Turbonomic generates only one SCALE action for a Cosmos DB document collection. Therefore, we only need to consider the
first item `0` in the actions array. The relevant value can be found in the details attribute.

 ```
+ details          = "Scale down RU for Document Collection example-doc-collection on cosmos from 1,000 to 400 in Example Azure Account"
 ```

Turbonomic recommends reducing the throughput from `1,000` RU/s to `400` RU/s. To implement this, we need to extract the
value `400` from the details attribute and convert it from text to a number. This can be done by introducing a local variable
in our Terraform code and using a regular expression along with a built-in Terraform function.

```terraform
locals {
  throughput_value = (
    tonumber(
      try(
    regex("\\bto\\s+(\\d+)", data.turbonomic_entity_actions.example.actions.0.details)[0], 400))
  )
}
```

-> **Note:** that we are still providing a default value of `400` in case we do not have any actions returned from Turbonomic.

With `throughput_value` now set to Turbonomic's recommended value, we can reference it in the `azurerm_cosmosdb_sql_container`
resource block to configure the throughput for our Cosmos DB container.

```terraform
resource "azurerm_cosmosdb_sql_container" "example" {
  name                = "example-doc-collection"
  resource_group_name = "example-resource-group"
  account_name        = "example-cosmos-account"
  database_name       = "example-cosmos-db"
  partition_key_paths = "/partitionKey"

  throughput = local.throughput_value

  indexing_policy {
    indexing_mode = "consistent"

    included_path {
      path = "/*"
    }

    excluded_path {
      path = "/\"_etag\"/?"
    }
  }
}
```

On future Terraform runs, Terraform will query Turbonomic and update `throughput_value` based on any new recommendations provided.


## ActionApiDTO Reference

For more information about the ActionApiDTO and its parameters, please refer to our [API Documentation](https://www.ibm.com/docs/en/tarm/8.17.0?topic=index-actionapidto)
