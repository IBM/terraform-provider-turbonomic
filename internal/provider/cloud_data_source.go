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
	"fmt"
	"regexp"
	"strings"

	turboclient "github.com/IBM/turbonomic-go-client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &cloudDataSource{}
	_ datasource.DataSourceWithConfigure = &cloudDataSource{}
)

func NewCloudDataSource() datasource.DataSource {
	return &cloudDataSource{}
}

type cloudDataSource struct {
	client *turboclient.Client
}

type cloudModel struct {
	UUID                types.String `tfsdk:"entity_uuid"`
	Name                types.String `tfsdk:"entity_name"`
	EntityType          types.String `tfsdk:"entity_type"`
	CurrentInstanceType types.String `tfsdk:"current_instance_type"`
	NewInstanceType     types.String `tfsdk:"new_instance_type"`
	DefaultInstanceSize types.String `tfsdk:"default_size"`
}

func (d *cloudDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_entity_recommendation"
}

func (d *cloudDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The following example demonstrates the syntax for the `turbonomic_cloud_entity_recommendation` data source.",
		Attributes: map[string]schema.Attribute{
			"entity_uuid": schema.StringAttribute{
				MarkdownDescription: "Turbonomic UUID of the cloud entity",
				Computed:            true,
			},
			"entity_name": schema.StringAttribute{
				MarkdownDescription: "name of the cloud entity",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"entity_type": schema.StringAttribute{
				MarkdownDescription: "type of the cloud entity",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("VirtualMachine", "VirtualVolume", "Database", "DatabaseServer"),
				},
			},
			"current_instance_type": schema.StringAttribute{
				MarkdownDescription: "current tier of the cloud entity",
				Computed:            true,
			},
			"new_instance_type": schema.StringAttribute{
				MarkdownDescription: "recommended tier of the cloud entity",
				Computed:            true,
			},
			"default_size": schema.StringAttribute{
				MarkdownDescription: "default tier of the cloud entity",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(20),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[A-Za-z0-9.-]+$`),
						"must contain only alphanumeric, '.' or '-' characters",
					),
				},
			},
		},
	}
}

func (d *cloudDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*turboclient.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *turboclient.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *cloudDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state cloudModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	entityName, entityType := state.Name.ValueString(), state.EntityType.ValueString()
	entity, errDiag := GetEntitiesByNameAndType(d.client, entityName, entityType, "CLOUD", "")
	if errDiag != nil {
		tflog.Error(ctx, errDiag.Detail())
		resp.Diagnostics.AddError(errDiag.Summary(), errDiag.Detail())
		return
	} else if len(entity) == 0 {
		errDetail := fmt.Sprintf("Entity %s of type %s not found in Turbonomic instance", entityName, entityType)
		tflog.Warn(ctx, errDetail)

		// if entity doesn't exist, update new instance type
		state = d.populateNewType(state)

		state.EntityType = types.StringNull()

		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Entity id found: %s\n", entity[0].UUID))
	state.UUID = types.StringValue(entity[0].UUID)
	state.CurrentInstanceType = types.StringValue(entity[0].Template.DisplayName)

	actions, errDiag := GetActionsByEntityUUIDAndType(d.client, entity[0].UUID, "SCALE")
	if errDiag != nil {
		tflog.Error(ctx, errDiag.Detail())
		resp.Diagnostics.AddError(errDiag.Summary(), errDiag.Detail())
		return
	} else if len(actions) == 0 {
		tflog.Trace(ctx, fmt.Sprintf("No matching action found for entity id: %s\n", entity[0].UUID))
		state.NewInstanceType = types.StringValue(strings.ToLower(entity[0].Template.DisplayName))
	} else {
		tflog.Debug(ctx, fmt.Sprintf("Action id found: %d\n", actions[0].ActionID))
		state.NewInstanceType = types.StringValue(strings.ToLower(actions[0].NewEntity.DisplayName))
	}

	state = d.populateNewType(state)

	if err := TagEntity(d.client, state.UUID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error while tagging an entity", err.Error())
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// update new-instance-size with default-size if it's still empty
func (*cloudDataSource) populateNewType(state cloudModel) cloudModel {
	if (state.NewInstanceType.IsNull() || len(state.NewInstanceType.ValueString()) == 0) && !state.DefaultInstanceSize.IsNull() {
		state.NewInstanceType = types.StringValue(strings.ToLower(state.DefaultInstanceSize.ValueString()))
	}

	return state
}
