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
