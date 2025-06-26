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
	"strings"

	turboclient "github.com/IBM/turbonomic-go-client"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &rdsDataSource{}
	_ datasource.DataSourceWithConfigure = &rdsDataSource{}
)

func NewRDSDataSource() datasource.DataSource {
	return &rdsDataSource{}
}

type rdsDataSource struct {
	client *turboclient.Client
}

type RDSModel struct {
	UUID               types.String `tfsdk:"entity_uuid"`
	Name               types.String `tfsdk:"entity_name"`
	EntityType         types.String `tfsdk:"entity_type"`
	CurrentComputeTier types.String `tfsdk:"current_instance_class"`
	NewComputeTier     types.String `tfsdk:"new_instance_class"`
	CurrentStorageTier types.String `tfsdk:"current_storage_type"`
	NewStorageTier     types.String `tfsdk:"new_storage_type"`
	DefaultComputeTier types.String `tfsdk:"default_instance_class"`
	DefaultStorageTier types.String `tfsdk:"default_storage_type"`
}

func (d *rdsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_aws_db_instance"
}

func (d *rdsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The following example demonstrates the syntax for the `turbonomic_aws_db_instance` data source.",
		Attributes: map[string]schema.Attribute{
			"entity_uuid": schema.StringAttribute{
				MarkdownDescription: "Turbonomic UUID of the AWS RDS entity",
				Computed:            true,
			},
			"entity_name": schema.StringAttribute{
				MarkdownDescription: "name of the AWS RDS entity",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"entity_type": schema.StringAttribute{
				MarkdownDescription: "type of the AWS RDS entity",
				Computed:            true,
			},
			"current_instance_class": schema.StringAttribute{
				MarkdownDescription: "current instance class of the AWS RDS entity",
				Computed:            true,
			},
			"new_instance_class": schema.StringAttribute{
				MarkdownDescription: "recommended instance class of the AWS RDS entity",
				Computed:            true,
			},
			"current_storage_type": schema.StringAttribute{
				MarkdownDescription: "current storage type of the AWS RDS entity",
				Computed:            true,
			},
			"new_storage_type": schema.StringAttribute{
				MarkdownDescription: "recommended storage type of the AWS RDS entity",
				Computed:            true,
			},
			"default_instance_class": schema.StringAttribute{
				MarkdownDescription: "default instance class of the AWS RDS entity",
				Optional:            true,
			},
			"default_storage_type": schema.StringAttribute{
				MarkdownDescription: "default storage type of the AWS RDS entity",
				Optional:            true,
			},
		},
	}
}

func (d rdsDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.RequiredTogether(
			path.MatchRoot("default_instance_class"),
			path.MatchRoot("default_storage_type"),
		),
	}
}

func (d *rdsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *rdsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state RDSModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	searchReq := turboclient.SearchRequest{
		Name:             state.Name.ValueString(),
		EntityType:       "DatabaseServer",
		EnvironmentType:  "CLOUD",
		CloudType:        "AWS",
		CaseSensitive:    true,
		SearchParameters: map[string]string{"query_type": "EXACT"}}

	searchVM, err := d.client.SearchEntityByName(searchReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to search Turbonomic",
			err.Error(),
		)
		return
	}

	if len(searchVM) == 1 {
		state.UUID = types.StringValue(searchVM[0].UUID)
		state.EntityType = types.StringValue("DatabaseServer")

		newActionReq := turboclient.ActionsRequest{
			Uuid:        searchVM[0].UUID,
			ActionState: []string{"READY"},
			ActionType:  []string{"SCALE"}}

		entityActions, err := d.client.GetActionsByUUID(newActionReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to retrieve actions from Turbonomic",
				err.Error(),
			)
			return
		}

		db_tiers := strings.Split(searchVM[0].Template.DisplayName, "-")

		if len(db_tiers) != 2 {
			resp.Diagnostics.AddError(
				"Unable to parse RDS scale action",
				fmt.Sprintf("DB action details: %v", db_tiers),
			)
			return
		}

		state.CurrentComputeTier = types.StringValue(db_tiers[0])
		state.CurrentStorageTier = types.StringValue(db_tiers[1])

		if len(entityActions) == 1 {
			for _, action := range entityActions[0].CompoundActions {
				if action.CurrentEntity.ClassName == "ComputeTier" {
					state.CurrentComputeTier = types.StringValue(action.CurrentEntity.DisplayName)
					state.NewComputeTier = types.StringValue(action.NewEntity.DisplayName)
				} else if action.CurrentEntity.ClassName == "StorageTier" {
					state.CurrentStorageTier = types.StringValue(action.CurrentEntity.DisplayName)
					state.NewStorageTier = types.StringValue(action.NewEntity.DisplayName)
				}
			}

		} else if len(entityActions) <= 0 {
			state.NewComputeTier = state.CurrentComputeTier
			state.NewStorageTier = state.CurrentStorageTier

		} else {
			detailMsg := fmt.Sprintf("Entity %s of type %s returned more than one scaling action...this is unexpected",
				state.Name.ValueString(),
				state.EntityType.ValueString())
			resp.Diagnostics.AddError("More than one scale action found.", detailMsg)

			tflog.Debug(ctx, "Too many actions returned", map[string]any{"dtoResponse": entityActions})
			return
		}

		err = TagEntity(d.client, state.UUID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error while tagging an entity", err.Error())
		}

	} else {
		var (
			msg       string
			detailMsg string
		)
		if len(searchVM) <= 0 {
			tflog.Debug(ctx, "No entities found, setting from defaults if they exist", map[string]any{"dtoResponse": searchVM})

			if state.NewComputeTier.ValueString() == "" && state.DefaultComputeTier.String() != "" {
				state.NewComputeTier = state.DefaultComputeTier
			}

			if state.NewStorageTier.ValueString() == "" && state.DefaultStorageTier.String() != "" {
				state.NewStorageTier = state.DefaultStorageTier
			}

		} else {
			msg = "Multiple Entities with provided name found"
			detailMsg = fmt.Sprintf("Multiple Entities with the name %s of type %s found in Turbonomic instance",
				state.Name.ValueString(),
				"DatabaseServer")

			tflog.Debug(ctx, "Too many entities returned", map[string]any{"dtoResponse": searchVM})
			resp.Diagnostics.AddError(msg, detailMsg)
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
