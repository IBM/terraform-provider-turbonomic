// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/IBM/turbonomic-go-client/api/generated"
	v2 "github.com/IBM/turbonomic-go-client/v2"
)

var (
	_ resource.Resource                = &targetResource{}
	_ resource.ResourceWithConfigure   = &targetResource{}
	_ resource.ResourceWithImportState = &targetResource{}
)

func NewTargetResource() resource.Resource {
	return &targetResource{}
}

type targetResource struct {
	v2Client *v2.Client
}

type targetResourceModel struct {
	// Input fields
	Type                 types.String `tfsdk:"type"`
	Category             types.String `tfsdk:"category"`
	InputFields          types.Map    `tfsdk:"input_fields"`
	InputFieldsWo        types.Map    `tfsdk:"input_fields_wo"`
	InputFieldsWoVersion types.Int64  `tfsdk:"input_fields_wo_version"`

	// Computed fields
	UUID                    types.String `tfsdk:"uuid"`
	IsProbeRegistered       types.Bool   `tfsdk:"is_probe_registered"`
	HealthState             types.String `tfsdk:"health_state"`
	HealthRollupState       types.String `tfsdk:"health_rollup_state"`
	HealthErrorText         types.String `tfsdk:"health_error_text"`
	LastSuccessfulDiscovery types.String `tfsdk:"last_successful_discovery"`
	LastEditTime            types.String `tfsdk:"last_edit_time"`
	LastEditUser            types.String `tfsdk:"last_edit_user"`
}

func (r *targetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_target"
}

func (r *targetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Turbonomic target (probe). Targets tell Turbonomic what to discover and monitor - cloud accounts, hypervisors, Kubernetes clusters, ITSM systems, and more.",
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				Description: "Probe type. Must match a probe registered in your Turbonomic instance. Examples: AWS, Azure, Terraform, vCenter, Kubernetes.",
				Required:    true,
			},
			"category": schema.StringAttribute{
				Description: "Probe category. Examples: Hypervisor, Infrastructure as Code, Cloud Management, Cloud Native.",
				Optional:    true,
			},
			"input_fields": schema.MapAttribute{
				Description: "Non-sensitive target configuration fields. Keys are InputField names (e.g. displayName, terraformHostname), values are strings. These are stored in Terraform state.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"input_fields_wo": schema.MapAttribute{
				Description: "Sensitive credential fields. Keys are InputField names (e.g. hcpToken, password), values are strings. These are NEVER stored in Terraform state. Requires Terraform 1.10+ when using ephemeral values.",
				Optional:    true,
				WriteOnly:   true,
				Sensitive:   true,
				ElementType: types.StringType,
			},
			"input_fields_wo_version": schema.Int64Attribute{
				Description: "Increment this integer to force Terraform to re-send input_fields_wo credentials on the next apply. Use this to rotate secrets without storing them in state.",
				Optional:    true,
			},
			"uuid": schema.StringAttribute{
				Description: "The UUID of the target, set by Turbonomic on creation. Use this to reference the target in other resources.",
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
				Description: "The health state of the target including its derived targets. Values: NORMAL, MINOR, MAJOR, CRITICAL.",
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
		},
	}
}

func (r *targetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*providerData)
	if !ok {
		resp.Diagnostics.AddError(
			"unexpected resource configure type",
			fmt.Sprintf("Expected *providerData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.v2Client = data.V2Client

	if r.v2Client == nil {
		resp.Diagnostics.AddError(
			"v2 client not available",
			"The v2 client is required for target operations but is not available.",
		)
		return
	}
}

func (r *targetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan targetResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Write-only attributes are nullified in the plan by the framework before
	// ApplyResourceChange is called. Read them from Config, which preserves the
	// original values supplied by the user.
	var config targetResourceModel
	diags = req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	inputFields, err := buildInputFields(ctx, plan.InputFields, config.InputFieldsWo)
	if err != nil {
		resp.Diagnostics.AddError("error building input fields", err.Error())
		return
	}

	dto := generated.TargetApiDTO{
		Type:        plan.Type.ValueString(),
		InputFields: &inputFields,
	}
	if !plan.Category.IsNull() && !plan.Category.IsUnknown() {
		dto.Category = plan.Category.ValueStringPointer()
	}

	regularCount := 0
	if !plan.InputFields.IsNull() && !plan.InputFields.IsUnknown() {
		regularCount = len(plan.InputFields.Elements())
	}
	writeOnlyCount := 0
	if !config.InputFieldsWo.IsNull() && !config.InputFieldsWo.IsUnknown() {
		writeOnlyCount = len(config.InputFieldsWo.Elements())
	}

	tflog.Debug(ctx, "creating target", map[string]interface{}{
		"type":                  dto.Type,
		"input_fields_count":    regularCount,
		"input_fields_wo_count": writeOnlyCount,
		"total_input_fields":    len(inputFields),
	})

	created, err := r.v2Client.Targets().Create(ctx, dto, nil)
	if err != nil {
		resp.Diagnostics.AddError("error creating target", err.Error())
		return
	}

	tflog.Debug(ctx, "target created successfully", map[string]interface{}{
		"uuid":                ptrToString(created.Uuid),
		"is_probe_registered": ptrToBool(created.IsProbeRegistered),
	})

	mapTargetResponseToModel(ctx, created, &plan)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *targetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state targetResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	targetUUID := state.UUID.ValueString()
	tflog.Debug(ctx, "reading target", map[string]interface{}{
		"uuid": targetUUID,
	})

	detailLevel := generated.TargetDetailLevelHEALTH
	target, err := r.v2Client.Targets().Get(ctx, targetUUID, &generated.GetTargetParams{
		DetailLevel: &detailLevel,
	})
	if err != nil {
		// Check for 404 - remove from state to allow re-creation
		if isNotFoundError(err) {
			tflog.Debug(ctx, "target not found - removing from state", map[string]interface{}{
				"uuid": targetUUID,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("error reading target", err.Error())
		return
	}

	healthState := ""
	if target.HealthSummary != nil {
		healthState = string(target.HealthSummary.HealthState)
	}
	tflog.Debug(ctx, "target read successfully", map[string]interface{}{
		"uuid":                targetUUID,
		"health_state":        healthState,
		"is_probe_registered": ptrToBool(target.IsProbeRegistered),
	})

	mapTargetResponseToModel(ctx, target, &state)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *targetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan targetResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state targetResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Write-only attributes are nullified in the plan - read from Config instead.
	var config targetResourceModel
	diags = req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	targetUUID := state.UUID.ValueString()

	inputFields, err := buildInputFields(ctx, plan.InputFields, config.InputFieldsWo)
	if err != nil {
		resp.Diagnostics.AddError("error building input fields", err.Error())
		return
	}

	dto := generated.TargetApiDTO{
		Type:        plan.Type.ValueString(),
		InputFields: &inputFields,
	}
	if !plan.Category.IsNull() && !plan.Category.IsUnknown() {
		dto.Category = plan.Category.ValueStringPointer()
	}

	regularCount := 0
	if !plan.InputFields.IsNull() && !plan.InputFields.IsUnknown() {
		regularCount = len(plan.InputFields.Elements())
	}
	writeOnlyCount := 0
	if !config.InputFieldsWo.IsNull() && !config.InputFieldsWo.IsUnknown() {
		writeOnlyCount = len(config.InputFieldsWo.Elements())
	}

	versionChanged := !plan.InputFieldsWoVersion.Equal(state.InputFieldsWoVersion)

	tflog.Debug(ctx, "updating target", map[string]interface{}{
		"uuid":                            targetUUID,
		"input_fields_count":              regularCount,
		"input_fields_wo_count":           writeOnlyCount,
		"total_input_fields":              len(inputFields),
		"input_fields_wo_version":         plan.InputFieldsWoVersion.ValueInt64(),
		"input_fields_wo_version_changed": versionChanged,
	})

	updated, err := r.v2Client.Targets().Update(ctx, targetUUID, dto)
	if err != nil {
		resp.Diagnostics.AddError("error updating target", err.Error())
		return
	}

	tflog.Debug(ctx, "target updated successfully", map[string]interface{}{
		"uuid":                targetUUID,
		"is_probe_registered": ptrToBool(updated.IsProbeRegistered),
		"last_edit_time":      ptrToString(updated.LastEditTime),
		"last_edit_user":      ptrToString(updated.LastEditUser),
	})

	mapTargetResponseToModel(ctx, updated, &plan)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *targetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state targetResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	targetUUID := state.UUID.ValueString()
	tflog.Debug(ctx, "deleting target", map[string]interface{}{
		"uuid": targetUUID,
		"type": state.Type.ValueString(),
	})

	if err := r.v2Client.Targets().Delete(ctx, targetUUID); err != nil {
		if isNotFoundError(err) {
			return
		}
		resp.Diagnostics.AddError("error deleting target", err.Error())
		return
	}

	tflog.Debug(ctx, "target deleted successfully", map[string]interface{}{
		"uuid": targetUUID,
	})
}

func (r *targetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

// buildInputFields merges the regular and write-only input field maps into the
// []InputFieldApiDTO slice required by the API.
func buildInputFields(ctx context.Context, regular, writeOnly types.Map) ([]generated.InputFieldApiDTO, error) {
	var fields []generated.InputFieldApiDTO

	if !regular.IsNull() && !regular.IsUnknown() {
		elements := make(map[string]string)
		if diags := regular.ElementsAs(ctx, &elements, false); diags.HasError() {
			return nil, fmt.Errorf("failed to read input_fields: %s", diags[0].Detail())
		}
		for k, v := range elements {
			name := k
			value := v
			fields = append(fields, generated.InputFieldApiDTO{
				Name:  &name,
				Value: &value,
			})
		}
	}

	if !writeOnly.IsNull() && !writeOnly.IsUnknown() {
		elements := make(map[string]string)
		if diags := writeOnly.ElementsAs(ctx, &elements, false); diags.HasError() {
			return nil, fmt.Errorf("failed to read input_fields_wo: %s", diags[0].Detail())
		}
		for k, v := range elements {
			name := k
			value := v
			fields = append(fields, generated.InputFieldApiDTO{
				Name:  &name,
				Value: &value,
			})
		}
	}

	return fields, nil
}

// mapTargetResponseToModel maps a TargetApiDTO API response back to the Terraform state model.
//
// Null-preservation rule for input_fields: only update keys already present in the
// model's current map. Never add new keys introduced by the API - doing so would cause
// "provider produced inconsistent result" errors on subsequent plans.
func mapTargetResponseToModel(_ context.Context, resp *generated.TargetApiDTO, model *targetResourceModel) {
	if resp == nil {
		return
	}

	// Computed identity/audit fields
	if resp.Uuid != nil {
		model.UUID = types.StringPointerValue(resp.Uuid)
	}
	if resp.IsProbeRegistered != nil {
		model.IsProbeRegistered = types.BoolPointerValue(resp.IsProbeRegistered)
	}
	if resp.LastEditTime != nil {
		model.LastEditTime = types.StringPointerValue(resp.LastEditTime)
	} else {
		model.LastEditTime = types.StringNull()
	}
	if resp.LastEditUser != nil {
		model.LastEditUser = types.StringPointerValue(resp.LastEditUser)
	} else {
		model.LastEditUser = types.StringNull()
	}

	// Health summary (populated when detail_level=HEALTH)
	if resp.HealthSummary != nil {
		model.HealthState = types.StringValue(string(resp.HealthSummary.HealthState))
		model.HealthRollupState = types.StringValue(string(resp.HealthSummary.RollupState))
		if resp.HealthSummary.TimeOfLastSuccessfulDiscovery != nil {
			model.LastSuccessfulDiscovery = types.StringPointerValue(resp.HealthSummary.TimeOfLastSuccessfulDiscovery)
		} else {
			model.LastSuccessfulDiscovery = types.StringNull()
		}
	} else {
		model.HealthState = types.StringNull()
		model.HealthRollupState = types.StringNull()
		model.LastSuccessfulDiscovery = types.StringNull()
	}

	// Health error text from the per-target health object
	if resp.Health != nil && resp.Health.ErrorText != nil {
		model.HealthErrorText = types.StringPointerValue(resp.Health.ErrorText)
	} else {
		model.HealthErrorText = types.StringNull()
	}

	// input_fields: sync non-secret fields back to state using null-preservation.
	// Only update keys that are already present in the model map - never add new ones.
	if resp.InputFields != nil && !model.InputFields.IsNull() && !model.InputFields.IsUnknown() {
		apiByName := make(map[string]generated.InputFieldApiDTO, len(*resp.InputFields))
		for _, f := range *resp.InputFields {
			if f.Name != nil {
				apiByName[*f.Name] = f
			}
		}

		// Iterate the plan map and update only existing keys from non-secret API fields
		planElements := make(map[string]string)
		// Safe to ignore diags here: map was already validated on the way in
		_ = model.InputFields.ElementsAs(context.Background(), &planElements, false)

		updated := make(map[string]string, len(planElements))
		for k, v := range planElements {
			if apiField, ok := apiByName[k]; ok {
				isSecret := apiField.IsSecret != nil && *apiField.IsSecret
				if !isSecret && apiField.Value != nil {
					updated[k] = *apiField.Value
					continue
				}
			}
			// Key not in API response, or secret - preserve plan value
			updated[k] = v
		}

		newMap, diags := types.MapValueFrom(context.Background(), types.StringType, updated)
		if !diags.HasError() {
			model.InputFields = newMap
		}
	}
}

// isNotFoundError checks whether an error from the go-client is a 404.
// The client returns errors in the form "... failed with status 404: ..."
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	needle := fmt.Sprintf("status %d", http.StatusNotFound)
	s := err.Error()
	for i := 0; i <= len(s)-len(needle); i++ {
		if s[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
