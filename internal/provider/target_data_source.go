// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/IBM/turbonomic-go-client/api/generated"
	v2 "github.com/IBM/turbonomic-go-client/v2"
)

var _ datasource.DataSource = &targetDataSource{}
var _ datasource.DataSourceWithConfigure = &targetDataSource{}

func NewTargetDataSource() datasource.DataSource {
	return &targetDataSource{}
}

type targetDataSource struct {
	v2Client *v2.Client
}

// targetDataSourceModel is the root Terraform state model for the data source.
// All inputs are optional filters; targets is the computed result list.
type targetDataSourceModel struct {
	// Optional filters - all may be omitted to return all targets.
	TypeFilter     types.String `tfsdk:"type"`
	CategoryFilter types.String `tfsdk:"category"`

	// Computed result list.
	Targets []targetItemModel `tfsdk:"targets"`
}

// targetItemModel represents a single target in the result list.
type targetItemModel struct {
	UUID                    types.String `tfsdk:"uuid"`
	Type                    types.String `tfsdk:"type"`
	Category                types.String `tfsdk:"category"`
	IsProbeRegistered       types.Bool   `tfsdk:"is_probe_registered"`
	HealthState             types.String `tfsdk:"health_state"`
	HealthRollupState       types.String `tfsdk:"health_rollup_state"`
	HealthErrorText         types.String `tfsdk:"health_error_text"`
	LastSuccessfulDiscovery types.String `tfsdk:"last_successful_discovery"`
	LastEditTime            types.String `tfsdk:"last_edit_time"`
	LastEditUser            types.String `tfsdk:"last_edit_user"`
	InputFields             types.Map    `tfsdk:"input_fields"` // Non-secret input fields only - the API never returns secret values.
}

func (d *targetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_target"
}

func (d *targetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	targetItemSchema := schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Description: "The UUID of the target.",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "Probe type (e.g. AWS, vCenter, Kubernetes).",
				Computed:    true,
			},
			"category": schema.StringAttribute{
				Description: "Probe category (e.g. Public Cloud, Hypervisor).",
				Computed:    true,
			},
			"is_probe_registered": schema.BoolAttribute{
				Description: "Whether the associated probe is running and registered with the Turbonomic system.",
				Computed:    true,
			},
			"health_state": schema.StringAttribute{
				Description: "The health state of the target. Values: NORMAL, MINOR, MAJOR, CRITICAL.",
				Computed:    true,
			},
			"health_rollup_state": schema.StringAttribute{
				Description: "The health state of the target including its derived targets.",
				Computed:    true,
			},
			"health_error_text": schema.StringAttribute{
				Description: "Error text from the most recent validation or discovery operation, if any.",
				Computed:    true,
			},
			"last_successful_discovery": schema.StringAttribute{
				Description: "ISO-8601 timestamp of the last successful discovery.",
				Computed:    true,
			},
			"last_edit_time": schema.StringAttribute{
				Description: "Timestamp of the last configuration change.",
				Computed:    true,
			},
			"last_edit_user": schema.StringAttribute{
				Description: "The user who last edited the target.",
				Computed:    true,
			},
			"input_fields": schema.MapAttribute{
				Description: "Non-secret target configuration fields returned by the API. " +
					"Secret credential fields are never included - the Turbonomic API does not return them.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}

	resp.Schema = schema.Schema{
		Description: "Returns all Turbonomic targets (probes), optionally filtered by probe type and/or category. " +
			"No inputs are required - omitting both filters returns every target. " +
			"Secret credential fields are never included in the results.",
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				Description: "Filter results to targets whose probe type matches this value exactly (e.g. AWS, vCenter, Kubernetes). " +
					"Omit to include all probe types.",
				Optional: true,
			},
			"category": schema.StringAttribute{
				Description: "Filter results to targets whose category matches this value exactly (e.g. Public Cloud, Hypervisor). " +
					"Omit to include all categories.",
				Optional: true,
			},
			"targets": schema.ListNestedAttribute{
				Description:  "List of targets matching the supplied filters. Empty when no targets match.",
				Computed:     true,
				NestedObject: targetItemSchema,
			},
		},
	}
}

func (d *targetDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*providerData)
	if !ok {
		resp.Diagnostics.AddError(
			"unexpected data source configure type",
			fmt.Sprintf("Expected *providerData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.v2Client = data.V2Client

	if d.v2Client == nil {
		resp.Diagnostics.AddError(
			"v2 client not available",
			"The v2 client is required for target data source operations but is not available.",
		)
	}
}

func (d *targetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config targetDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	typeFilter := config.TypeFilter.ValueString()
	categoryFilter := config.CategoryFilter.ValueString()

	tflog.Debug(ctx, "reading targets data source", map[string]interface{}{
		"type_filter":     typeFilter,
		"category_filter": categoryFilter,
	})

	detailLevel := generated.TargetDetailLevelHEALTH
	all, _, err := d.v2Client.Targets().List(ctx, &generated.GetTargetsParams{
		DetailLevel: &detailLevel,
	})
	if err != nil {
		resp.Diagnostics.AddError("error listing targets", err.Error())
		return
	}

	var items []targetItemModel
	for i := range all {
		t := &all[i]

		if typeFilter != "" && t.Type != typeFilter {
			continue
		}
		if categoryFilter != "" && (t.Category == nil || *t.Category != categoryFilter) {
			continue
		}

		items = append(items, mapTargetDTOToItemModel(t))
	}

	// Ensure we always write a non-nil list so state is consistent.
	if items == nil {
		items = []targetItemModel{}
	}

	tflog.Debug(ctx, "targets data source read complete", map[string]interface{}{
		"total_fetched": len(all),
		"matched":       len(items),
		"type_filter":   typeFilter,
	})

	state := targetDataSourceModel{
		TypeFilter:     config.TypeFilter,
		CategoryFilter: config.CategoryFilter,
		Targets:        items,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// mapTargetDTOToItemModel converts a single TargetApiDTO into a targetItemModel.
// Secret input fields (isSecret=true) are never included - the API does not
// return their values and they must not appear in data source state.
func mapTargetDTOToItemModel(dto *generated.TargetApiDTO) targetItemModel {
	item := targetItemModel{}

	if dto.Uuid != nil {
		item.UUID = types.StringPointerValue(dto.Uuid)
	} else {
		item.UUID = types.StringNull()
	}
	item.Type = types.StringValue(dto.Type)
	if dto.Category != nil {
		item.Category = types.StringPointerValue(dto.Category)
	} else {
		item.Category = types.StringNull()
	}
	if dto.IsProbeRegistered != nil {
		item.IsProbeRegistered = types.BoolPointerValue(dto.IsProbeRegistered)
	} else {
		item.IsProbeRegistered = types.BoolNull()
	}
	if dto.LastEditTime != nil {
		item.LastEditTime = types.StringPointerValue(dto.LastEditTime)
	} else {
		item.LastEditTime = types.StringNull()
	}
	if dto.LastEditUser != nil {
		item.LastEditUser = types.StringPointerValue(dto.LastEditUser)
	} else {
		item.LastEditUser = types.StringNull()
	}

	if dto.HealthSummary != nil {
		item.HealthState = types.StringValue(string(dto.HealthSummary.HealthState))
		item.HealthRollupState = types.StringValue(string(dto.HealthSummary.RollupState))
		if dto.HealthSummary.TimeOfLastSuccessfulDiscovery != nil {
			item.LastSuccessfulDiscovery = types.StringPointerValue(dto.HealthSummary.TimeOfLastSuccessfulDiscovery)
		} else {
			item.LastSuccessfulDiscovery = types.StringNull()
		}
	} else {
		item.HealthState = types.StringNull()
		item.HealthRollupState = types.StringNull()
		item.LastSuccessfulDiscovery = types.StringNull()
	}

	if dto.Health != nil && dto.Health.ErrorText != nil {
		item.HealthErrorText = types.StringPointerValue(dto.Health.ErrorText)
	} else {
		item.HealthErrorText = types.StringNull()
	}

	fieldMap := make(map[string]string)
	if dto.InputFields != nil {
		for _, f := range *dto.InputFields {
			if f.Name == nil || (f.IsSecret != nil && *f.IsSecret) {
				continue
			}
			if f.Value != nil {
				fieldMap[*f.Name] = *f.Value
			}
		}
	}
	m, diags := types.MapValueFrom(context.Background(), types.StringType, fieldMap)
	if !diags.HasError() {
		item.InputFields = m
	}

	return item
}
