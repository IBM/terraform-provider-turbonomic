// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS-IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provider

import (
	"context"
	"os"

	turboclient "github.com/IBM/turbonomic-go-client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ provider.Provider = &TurbonomicProvider{}

type TurbonomicProvider struct {
	version string
}

type TurbonomicProviderModel struct {
	Hostname   types.String `tfsdk:"hostname"`
	Username   types.String `tfsdk:"username"`
	Password   types.String `tfsdk:"password"`
	Skipverify types.Bool   `tfsdk:"skipverify"`
}

func (p *TurbonomicProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "turbonomic"
	resp.Version = p.version
}

func (p *TurbonomicProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with a Turbonomic instance.",
		Attributes: map[string]schema.Attribute{
			"hostname": schema.StringAttribute{
				MarkdownDescription: "Hostname or IP Address of Turbonomic Instance",
				Description:         "Hostname or IP Address of Turbonomic Instance",
				Optional:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Username to access the Turbonomic Instance",
				Description:         "Username to access the Turbonomic Instance",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password for the username to access the Turbonomic Instance",
				Description:         "Password for the username to access the Turbonomic Instance",
				Optional:            true,
				Sensitive:           true,
			},
			"skipverify": schema.BoolAttribute{
				MarkdownDescription: "Boolean on whether to verify the SSL/TLS certificate for the hostname",
				Description:         "Boolean on whether to verify the SSL/TLS certificate for the hostname",
				Optional:            true,
			},
		},
	}
}

func (p *TurbonomicProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring Turbonomic client")

	var config TurbonomicProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Hostname.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("hostname"),
			"Unknown Turbonomic API Hostname",
			"The provider cannot create the Turbonomic API client; unknown configuration value for the Turbonomic API hostname. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the TURBO_HOSTNAME environment variable.",
		)
	}

	if config.Username.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Unknown Turbonomic API Username",
			"The provider cannot create the Turbonomic API client; unknown configuration value for the Turbonomic API username. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the TURBO_USERNAME environment variable.",
		)
	}

	if config.Password.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Unknown Turbonomic API Password",
			"The provider cannot create the Turbonomic API client; unknown configuration value for the Turbonomic API password. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the TURBO_PASSWORD environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	hostname := os.Getenv("TURBO_HOSTNAME")
	username := os.Getenv("TURBO_USERNAME")
	password := os.Getenv("TURBO_PASSWORD")
	skipverify := false

	if !config.Hostname.IsNull() {
		hostname = config.Hostname.ValueString()
	}

	if !config.Username.IsNull() {
		username = config.Username.ValueString()
	}

	if !config.Password.IsNull() {
		password = config.Password.ValueString()
	}

	if !config.Skipverify.IsNull() {
		skipverify = config.Skipverify.ValueBool()
	}

	if hostname == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("hostname"),
			"Missing Turbonomic API Hostname",
			"The provider cannot create the Turbonomic API client; missing or empty value for the Turbonomic API hostname. "+
				"Set the hostname value in the configuration or use the TURBO_HOSTNAME environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if username == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Missing Turbonomic API Username",
			"The provider cannot create the Turbonomic API client; missing or empty value for the Turbonomic API username. "+
				"Set the username value in the configuration or use the TURBO_USERNAME environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if password == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Missing Turbonomic API Password",
			"The provider cannot create the Turbonomic API client; missing or empty value for the Turbonomic API password. "+
				"Set the password value in the configuration or use the TURBO_PASSWORD environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Create a new Turbonomic client using the configuration values
	newClientOpts := turboclient.ClientParameters{Hostname: hostname, Username: username, Password: password, Skipverify: skipverify}
	client, err := turboclient.NewClient(&newClientOpts)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Turbonomic API Client",
			"An unexpected error occurred when creating the Turbonomic API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Turbonomic Client Error: "+err.Error(),
		)
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *TurbonomicProvider) Resources(ctx context.Context) []func() resource.Resource {
	return nil
}

func (p *TurbonomicProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewCloudDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &TurbonomicProvider{
			version: version,
		}
	}
}
