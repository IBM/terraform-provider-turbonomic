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
	_ datasource.DataSource              = &azureManagedDiskDataSource{}
	_ datasource.DataSourceWithConfigure = &azureManagedDiskDataSource{}
)

func NewAzureManagedDiskDataSource() datasource.DataSource {
	return &azureManagedDiskDataSource{}
}

type AzureManagedDiskModel struct {
	UUID                  types.String `tfsdk:"entity_uuid"`
	Name                  types.String `tfsdk:"entity_name"`
	EntityType            types.String `tfsdk:"entity_type"`
	CurrentStorageAccType types.String `tfsdk:"current_storage_account_type"`
	NewStorageAccType     types.String `tfsdk:"new_storage_account_type"`
	DefaultStorageAccType types.String `tfsdk:"default_storage_account_type"`
}

type DiskType string

const (
	StandardHDD      DiskType = "Standard HDD"
	StandardSSD      DiskType = "Standard SSD"
	PremiumSSD       DiskType = "Premium SSD"
	UltraDisk        DiskType = "Ultra Disk"
	PremiumSSDv2     DiskType = "Premium SSD v2"
	ZoneRedundantSSD DiskType = "Zone-redundant SSD"
)

var (
	diskTypeToStorageType = map[DiskType]string{
		StandardHDD:      "Standard_LRS",
		StandardSSD:      "StandardSSD_LRS",
		PremiumSSD:       "Premium_LRS",
		UltraDisk:        "UltraSSD_LRS",
		PremiumSSDv2:     "PremiumV2_LRS",
		ZoneRedundantSSD: "StandardSSD_ZRS",
	}
	azure_storage_tiers = slices.Collect(maps.Values(diskTypeToStorageType))
)

type azureManagedDiskDataSource struct {
	client *turboclient.Client
}

func (d *azureManagedDiskDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_azurerm_managed_disk"
}

func (d *azureManagedDiskDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The following example demonstrates the syntax for the `turbonomic_azurerm_managed_disk` data source.",
		Attributes: map[string]schema.Attribute{
			"entity_uuid": schema.StringAttribute{
				MarkdownDescription: "Turbonomic UUID of the Azure Managed Disk entity",
				Computed:            true,
			},
			"entity_name": schema.StringAttribute{
				MarkdownDescription: "name of the Azure Managed Disk entity",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"entity_type": schema.StringAttribute{
				MarkdownDescription: "type of the Azure Managed Disk entity",
				Computed:            true,
			},
			"current_storage_account_type": schema.StringAttribute{
				MarkdownDescription: "current storage type of the Azure Managed Disk entity",
				Computed:            true,
			},
			"new_storage_account_type": schema.StringAttribute{
				MarkdownDescription: "recommended storage type of the Azure Managed Disk entity",
				Computed:            true,
			},
			"default_storage_account_type": schema.StringAttribute{
				MarkdownDescription: "default storage type of the Azure Managed Disk entity",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(azure_storage_tiers...),
				},
			},
		},
	}
}

func (d *azureManagedDiskDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *azureManagedDiskDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state AzureManagedDiskModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	searchReq := turboclient.SearchRequest{
		Name:             state.Name.ValueString(),
		EntityType:       "VirtualVolume",
		EnvironmentType:  "CLOUD",
		CloudType:        "AZURE",
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
		state.EntityType = types.StringValue("VirtualVolume")

		newActionReq := turboclient.ActionsRequest{
			Uuid:        searchVM[0].UUID,
			ActionState: []string{"READY"},
			ActionType:  []string{"SCALE"}}

		storageType, err := getStorageType(searchVM[0].Template.DisplayName)

		if err != nil {
			resp.Diagnostics.AddError(
				"Unknown storage type for current value",
				err.Error(),
			)
			return
		}

		state.CurrentStorageAccType = types.StringValue(storageType)
		entityActions, err := d.client.GetActionsByUUID(newActionReq)

		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to retrieve actions from Turbonomic",
				err.Error(),
			)
			return
		}

		if len(entityActions) == 1 {
			storageType, err := getStorageType(entityActions[0].NewEntity.DisplayName)

			if err != nil {
				resp.Diagnostics.AddError(
					"Unknown storage type for new value",
					err.Error(),
				)
				return
			}

			state.NewStorageAccType = types.StringValue(storageType)

		} else if len(entityActions) <= 0 {
			state.NewStorageAccType = state.CurrentStorageAccType
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
		if len(searchVM) <= 0 {
			tflog.Debug(ctx, "No entities found, setting from defaults if they exist", map[string]any{"dtoResponse": searchVM})

			if state.NewStorageAccType.ValueString() == "" && state.DefaultStorageAccType.String() != "" {
				state.NewStorageAccType = state.DefaultStorageAccType
			}

		} else {
			var (
				msg       string
				detailMsg string
			)
			msg = "Multiple Entities with provided name found"
			detailMsg = fmt.Sprintf("Multiple Entities with the name %s of type %s found in Turbonomic instance",
				state.Name.ValueString(),
				"VirtualVolume")

			tflog.Debug(ctx, "Too many entities returned", map[string]any{"dtoResponse": searchVM})
			resp.Diagnostics.AddError(msg, detailMsg)
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func getStorageType(disk string) (string, error) {
	storageType, ok := diskTypeToStorageType[DiskType(
		strings.TrimPrefix(
			disk, "Managed ",
		),
	)]
	if !ok {
		return "", fmt.Errorf("unknown storage type provided: %s", disk)
	}
	return storageType, nil
}
