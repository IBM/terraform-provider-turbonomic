# IBM Turbonomic Terraform Provider

Use this repository to build the IBM Turbonomic Terraform Provider, which supplies data resources to interact with the Turbonomic API. For more information about the provider, see the [documentation](https://registry.terraform.io/providers/IBM/turbonomic/latest/docs).

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.5.7
- [Go](https://golang.org/doc/install) >= 1.23.7

## Building The Provider

1. Clone this repository.
1. Go to the `terraform-provider-turbonomic` directory.
1. Build the provider by running the Go `install` command:

```shell
go install
```

## Using the provider

 To get started using the Turbonomic provider, see the [documentation](https://registry.terraform.io/providers/IBM/turbonomic/latest/docs).
