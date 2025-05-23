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
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	turboclient "github.com/IBM/turbonomic-go-client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ provider.Provider = &turbonomicProvider{}

type turbonomicProvider struct {
	version  string
	typeName string
}

type TurbonomicProviderModel struct {
	Hostname     types.String `tfsdk:"hostname"`
	Username     types.String `tfsdk:"username"`
	Password     types.String `tfsdk:"password"`
	ClientId     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	Role         types.String `tfsdk:"role"`
	Skipverify   types.Bool   `tfsdk:"skipverify"`
}

func (p *turbonomicProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = p.typeName
	resp.Version = p.version
}

func (p *turbonomicProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with a Turbonomic instance.",
		Attributes: map[string]schema.Attribute{
			"hostname": schema.StringAttribute{
				MarkdownDescription: "hostname or IP Address of Turbonomic Instance; " +
					"use TURBO_HOSTNAME to set with an enviornment variable",
				Description: "hostname or IP Address of Turbonomic Instance; " +
					"use TURBO_HOSTNAME to set with an enviornment variable",
				Optional: true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "username to access the Turbonomic Instance; " +
					"use TURBO_USERNAME to set with an enviornment variable",
				Description: "username to access the Turbonomic Instance; " +
					"use TURBO_USERNAME to set with an enviornment variable",
				Optional: true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "password for the username to access the Turbonomic Instance; " +
					"use TURBO_PASSWORD to set with an enviornment variable",
				Description: "password for the username to access the Turbonomic Instance; " +
					"use TURBO_PASSWORD to set with an enviornment variable",
				Optional:  true,
				Sensitive: true,
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "the OAuth 2.0 client ID that can be used to access the Turbonomic " +
					"instance; use TURBO_CLIENT_ID to set with an enviornment variable",
				Description: "the OAuth 2.0 client ID that can be used to access the Turbonomic " +
					"instance; use TURBO_CLIENT_ID to set with an enviornment variable",
				Optional: true,
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "the OAuth 2.0 client secret that can be used to access the Turbonomic " +
					"instance; use TURBO_CLIENT_SECRET to set with an enviornment variable",
				Description: "the OAuth 2.0 client secret that can be used to access the Turbonomic " +
					"instance; use TURBO_CLIENT_SECRET to set with an enviornment variable",
				Optional:  true,
				Sensitive: true,
			},
			"role": schema.StringAttribute{
				MarkdownDescription: "the OAuth 2.0 role that can be used to access the Turbonomic " +
					"instance; use TURBO_ROLE to set with an enviornment variable",
				Description: "the OAuth 2.0 role that can be used to access the Turbonomic " +
					"instance; use TURBO_ROLE to set with an enviornment variable",
				Optional: true,
			},
			"skipverify": schema.BoolAttribute{
				MarkdownDescription: "boolean on whether to verify the SSL or TLS certificate for the hostname",
				Description:         "boolean on whether to verify the SSL or TLS certificate for the hostname",
				Optional:            true,
			},
		},
	}
}

func (p *turbonomicProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
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
			"Unknown Turbonomic API hostname",
			"The provider cannot create the Turbonomic API client; unknown configuration value for the Turbonomic API hostname. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the TURBO_HOSTNAME environment variable.",
		)
	}

	if config.Username.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Unknown Turbonomic API username",
			"The provider cannot create the Turbonomic API client; unknown configuration value for the Turbonomic API username. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the TURBO_USERNAME environment variable.",
		)
	}

	if config.Password.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Unknown Turbonomic API password",
			"The provider cannot create the Turbonomic API client; unknown configuration value for the Turbonomic API password. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the TURBO_PASSWORD environment variable.",
		)
	}

	if config.ClientId.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("client_id"),
			"Unknown Turbonomic API client_id",
			"The provider cannot create the Turbonomic API client; unknown configuration value for the Turbonomic API client_id. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the TURBO_CLIENT_ID environment variable.",
		)
	}

	if config.ClientSecret.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("client_secret"),
			"Unknown Turbonomic API client_secret",
			"The provider cannot create the Turbonomic API client; unknown configuration value for the Turbonomic API client_secret. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the TURBO_CLIENT_SECRET environment variable.",
		)
	}

	if config.Role.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("role"),
			"Unknown Turbonomic API role",
			"The provider cannot create the Turbonomic API client; unknown configuration value for the Turbonomic API role. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the TURBO_ROLE environment variable.",
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
	clientId := os.Getenv("TURBO_CLIENT_ID")
	clientSecret := os.Getenv("TURBO_CLIENT_SECRET")
	role := os.Getenv("TURBO_ROLE")
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

	if !config.ClientId.IsNull() {
		clientId = config.ClientId.ValueString()
	}

	if !config.ClientSecret.IsNull() {
		clientSecret = config.ClientSecret.ValueString()
	}

	if !config.Role.IsNull() {
		role = config.Role.ValueString()
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

	if len(StringsWithValues(username, clientId)) != 1 {
		resp.Diagnostics.AddError(
			"Invalid Attribute Combination -> Multiple Authentication Methods Provided",
			"Exactly one of these attributes must be configured: [username, client_id]",
		)
	}

	if stringsValues := StringsWithValues(username, password); len(stringsValues) != 0 &&
		len(stringsValues) != 2 {

		resp.Diagnostics.AddError(
			"Invalid Attribute Combination -> Username/Password",
			"These attributes must be configured together: [username, password]",
		)
	}

	if stringsValues := StringsWithValues(clientId, clientSecret, role); len(stringsValues) != 0 &&
		len(stringsValues) != 3 {

		resp.Diagnostics.AddError(
			"Invalid Attribute Combination -> oAuth",
			"These attributes must be configured together: [client_id, client_secret, role]",
		)
	}

	if role != "" {
		validRoles := []string{"ADMINISTRATOR", "SITE_ADMIN", "AUTOMATOR",
			"DEPLOYER", "ADVISOR", "OBSERVER", "OPERATIONAL_OBSERVER", "SHARED_ADVISOR",
			"SHARED_OBSERVER", "REPORT_EDITOR"}
		validRolefmt, _ := json.Marshal(validRoles)
		if !slices.Contains(validRoles, role) {
			msg := fmt.Sprintf("Attribute role value must be one of: %s, got: %s", validRolefmt, role)
			resp.Diagnostics.AddError(
				"Invalid Attribute Value Match -> Unknown Role",
				msg,
			)
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Create a new Turbonomic client using the configuration values

	var newClientOpts turboclient.ClientParameters
	if username != "" {
		newClientOpts = turboclient.ClientParameters{Hostname: hostname, Username: username, Password: password, Skipverify: skipverify}

	} else if clientId != "" {
		newClientOpts = turboclient.ClientParameters{
			Hostname: hostname,
			OAuthCreds: turboclient.OAuthCreds{
				ClientId:     clientId,
				ClientSecret: clientSecret,
				Role:         turboclient.GetRolefromString(strings.ToUpper(role))},
			Skipverify: skipverify}
	} else {
		resp.Diagnostics.AddError(
			"Unable to Create Turbonomic API Client",
			"No credentials have been passed to the client. This should have been "+
				"caught by the validators",
		)
		return
	}

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

func StringsWithValues(ss ...string) []string {
	var populatedStrings []string

	for _, s := range ss {
		if s != "" {
			populatedStrings = append(populatedStrings, s)
		}
	}
	return populatedStrings
}

func (p *turbonomicProvider) Resources(ctx context.Context) []func() resource.Resource {
	return nil
}

func (p *turbonomicProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewCloudDataSource,
	}
}

func New(version, typeName string) func() provider.Provider {
	return func() provider.Provider {
		return &turbonomicProvider{
			version:  version,
			typeName: typeName,
		}
	}
}
