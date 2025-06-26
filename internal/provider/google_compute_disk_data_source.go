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
	"maps"
	"slices"

	turboclient "github.com/IBM/turbonomic-go-client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	google_storage_tiers_map = map[string]string{
		"Standard Persistent Disk": "pd-standard",
		"Balanced Persistent Disk": "pd-balanced",
		"SSD Persistent Disk":      "pd-ssd",
		"Extreme Persistent Disk":  "pd-extreme",
		"Hyperdisk Balanced":       "hyperdisk-balanced",
		"Hyperdisk Throughput":     "hyperdisk-throughput",
		"Hyperdisk Extreme":        "hyperdisk-extreme",
	}

	google_storage_tiers = slices.Collect(maps.Values(google_storage_tiers_map))
)

var (
	_ datasource.DataSource              = &googleComputeDiskDataSource{}
	_ datasource.DataSourceWithConfigure = &googleComputeDiskDataSource{}
)

func NewGoogleComputeDiskDataSource() datasource.DataSource {
	return &googleComputeDiskDataSource{}
}

type googleComputeDiskDataSource struct {
	client *turboclient.Client
}

type googleComputeDiskModel struct {
	UUID        types.String `tfsdk:"entity_uuid"`
	Name        types.String `tfsdk:"entity_name"`
	EntityType  types.String `tfsdk:"entity_type"`
	CurrentType types.String `tfsdk:"current_type"`
	NewType     types.String `tfsdk:"new_type"`
	DefaultType types.String `tfsdk:"default_type"`
}

func (d *googleComputeDiskDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_google_compute_disk"
}

func (d *googleComputeDiskDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The following example demonstrates the syntax for the `turbonomic_google_compute_disk` data source.",
		Attributes: map[string]schema.Attribute{
			"entity_uuid": schema.StringAttribute{
				MarkdownDescription: "Turbonomic UUID of the virtual volume entity",
				Computed:            true,
			},
			"entity_name": schema.StringAttribute{
				MarkdownDescription: "name of the virtual volume entity",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"entity_type": schema.StringAttribute{
				MarkdownDescription: "type of the virtual volume entity",
				Computed:            true,
			},
			"current_type": schema.StringAttribute{
				MarkdownDescription: "current tier of the virtual volume entity",
				Computed:            true,
			},
			"new_type": schema.StringAttribute{
				MarkdownDescription: "recommended tier of the virtual volume entity",
				Computed:            true,
			},
			"default_type": schema.StringAttribute{
				MarkdownDescription: "default tier of the virtual volume entity",
				Validators:          []validator.String{stringvalidator.OneOf(google_storage_tiers...)},
				Optional:            true,
			},
		},
	}
}

func (d *googleComputeDiskDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *googleComputeDiskDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state googleComputeDiskModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	searchReq := turboclient.SearchRequest{
		Name:             state.Name.ValueString(),
		EntityType:       "VirtualVolume",
		EnvironmentType:  "CLOUD",
		CloudType:        "GCP",
		CaseSensitive:    true,
		SearchParameters: map[string]string{"query_type": "EXACT"}}

	searchEntity, err := d.client.SearchEntityByName(searchReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to search Turbonomic",
			err.Error(),
		)
		return
	}

	if len(searchEntity) == 1 {
		state.UUID = types.StringValue(searchEntity[0].UUID)
		state.CurrentType = types.StringValue(google_storage_tiers_map[searchEntity[0].Template.DisplayName])
		state.EntityType = types.StringValue("VirtualVolume")

		newActionReq := turboclient.ActionsRequest{
			Uuid:        searchEntity[0].UUID,
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

		if len(entityActions) == 1 {
			state.NewType = types.StringValue(google_storage_tiers_map[entityActions[0].NewEntity.DisplayName])
		} else if len(entityActions) <= 0 {
			state.NewType = state.CurrentType
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
		if len(searchEntity) <= 0 {
			tflog.Debug(ctx, "No entities found, setting from default if it exists", map[string]any{"dtoResponse": searchEntity})

			if state.NewType.ValueString() == "" && state.DefaultType.String() != "" {
				state.NewType = state.DefaultType
			}

		} else {
			msg = "Multiple Entities with provided name found"
			detailMsg = fmt.Sprintf("Multiple Entities with the name %s of type %s found in Turbonomic instance",
				state.Name.ValueString(),
				"VirtualVolume")

			tflog.Debug(ctx, "Too many entities returned", map[string]any{"dtoResponse": searchEntity})
			resp.Diagnostics.AddError(msg, detailMsg)
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
