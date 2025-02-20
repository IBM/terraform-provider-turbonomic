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

	turboclient "github.com/IBM/turbonomic-go-client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &CloudDataSource{}
	_ datasource.DataSourceWithConfigure = &CloudDataSource{}
)

func NewCloudDataSource() datasource.DataSource {
	return &CloudDataSource{}
}

type CloudDataSource struct {
	client *turboclient.Client
}

type CloudModel struct {
	UUID                types.String `tfsdk:"entity_uuid"`
	Name                types.String `tfsdk:"entity_name"`
	EntityType          types.String `tfsdk:"entity_type"`
	CurrentInstanceType types.String `tfsdk:"current_instance_type"`
	NewInstanceType     types.String `tfsdk:"new_instance_type"`
}

func (d *CloudDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_entity_recommendation"
}

func (d *CloudDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The following example demonstrates the syntax for the `turbonomic_cloud_entity_recommendation` data source.",
		Attributes: map[string]schema.Attribute{
			"entity_uuid": schema.StringAttribute{
				MarkdownDescription: "Turbonomic UUID of the cloud entity",
				Computed:            true,
			},
			"entity_name": schema.StringAttribute{
				MarkdownDescription: "Name of the cloud entity",
				Required:            true,
			},
			"entity_type": schema.StringAttribute{
				MarkdownDescription: "Type of the cloud entity",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("VirtualMachine", "VirtualVolume", "Database", "DatabaseServer"),
				},
			},
			"current_instance_type": schema.StringAttribute{
				MarkdownDescription: "Current tier of the cloud entity",
				Computed:            true,
			},
			"new_instance_type": schema.StringAttribute{
				MarkdownDescription: "Recommended tier of the cloud entity",
				Computed:            true,
			},
		},
	}
}

func (d *CloudDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *CloudDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state CloudModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	searchReq := turboclient.SearchRequest{
		Name:             state.Name.ValueString(),
		EntityType:       state.EntityType.ValueString(),
		EnvironmentType:  "CLOUD",
		CaseSensitive:    true,
		SearchParameters: map[string]string{"query_type": "EXACT"}}

	searchVM, err := d.client.SearchEntityByName(searchReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Search Turbonomic",
			err.Error(),
		)
		return
	}

	if len(searchVM) == 1 {
		state.UUID = types.StringValue(searchVM[0].UUID)
		state.CurrentInstanceType = types.StringValue(searchVM[0].Template.DisplayName)
		newActionReq := turboclient.ActionsRequest{
			Uuid:        searchVM[0].UUID,
			ActionState: []string{"READY"},
			ActionType:  []string{"SCALE"}}

		entityActions, err := d.client.GetActionsByUUID(newActionReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Retrive actions from Turbonomic",
				err.Error(),
			)
			return
		}

		if len(entityActions) == 1 {
			state.NewInstanceType = types.StringValue(entityActions[0].NewEntity.DisplayName)
		} else if len(entityActions) <= 0 {
			state.NewInstanceType = types.StringValue(searchVM[0].Template.DisplayName)
		} else {
			detailMsg := fmt.Sprintf("Entitiy %s of type %s returned more than one scaling action...this is unexpected",
				state.Name.ValueString(),
				state.EntityType.ValueString())
			resp.Diagnostics.AddError("More than one scale action found.", detailMsg)

			tflog.Debug(ctx, "Too many actions returned", map[string]any{"dtoResponse": entityActions})
			return
		}

	} else {
		var (
			msg       string
			detailMsg string
		)
		if len(searchVM) <= 0 {
			msg = "Entitiy not found in Turbonomic instance"
			detailMsg = fmt.Sprintf("Entitiy %s of type %s not found in Turbonomic instance",
				state.Name.ValueString(),
				state.EntityType.ValueString())

			tflog.Debug(ctx, detailMsg)
		} else {
			msg = "Multiple Entities with provided name found"
			detailMsg = fmt.Sprintf("Multiple Entities with the name %s of type %s found in Turbonomic instance",
				state.Name.ValueString(),
				state.EntityType.ValueString())

			tflog.Debug(ctx, "Too many entities returned", map[string]any{"dtoResponse": searchVM})
			resp.Diagnostics.AddError(msg, detailMsg)
			return
		}

	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
