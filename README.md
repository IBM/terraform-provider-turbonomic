# IBM Turbonomic Terraform Provider

Use this repository to build the IBM Turbonomic Terraform Provider, which supplies data resources to interact with the Turbonomic API. For more information about the provider, see the [documentation](https://registry.terraform.io/providers/IBM/turbonomic/latest/docs).

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.8.5
- [Go](https://golang.org/doc/install) >= 1.23.7

## Building the Provider

1. Clone this repository.
2. Go to the `terraform-provider-turbonomic` directory.
3. Update `providerConfig` and `vmName` in `internal/provider/cloud_data_source_test.go`.
4. Build the provider by running the `make build` command / or run command `go build -o terraform-provider-turbonomic` directly to skip tests.

## Using the Provider

 To get started using the Turbonomic provider, see the [documentation](https://registry.terraform.io/providers/IBM/turbonomic/latest/docs).
