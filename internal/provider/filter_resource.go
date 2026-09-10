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

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/google/uuid"
)

var (
	_ resource.Resource                = &filterResource{}
	_ resource.ResourceWithModifyPlan  = &filterResource{}
	_ resource.ResourceWithImportState = &filterResource{}
)

// NewFilterResource returns a new turbonomic_filter resource instance.
func NewFilterResource() resource.Resource {
	return &filterResource{}
}

// filterResource is a local state-only resource. No Turbonomic API calls are made.
// It stores a named set of criteria that can be referenced by other resources
// (e.g. turbonomic_group) to avoid repeating the same filter definitions.
type filterResource struct{}

// filterResourceModel is the Terraform state model for turbonomic_filter.
type filterResourceModel struct {
	// Input
	ID           types.String `tfsdk:"id"`
	CriteriaList types.List   `tfsdk:"criteria_list"` // list of criteriaListModel objects

	// Computed output - same list re-exported as an attribute so callers can reference it
	// directly in expressions (e.g. concat, dynamic blocks).
	Criteria types.List `tfsdk:"criteria"`
}

func (r *filterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_filter"
}

func (r *filterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A local, state-only resource that stores a named set of filter criteria. " +
			"No Turbonomic API calls are made. " +
			"Use the `criteria` output to reference this filter's criteria in other resources " +
			"(e.g. via `dynamic \"criteria_list\"` in a `turbonomic_group`, or combined with " +
			"`concat()` from multiple filters).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "A stable UUID assigned on creation. Used as a stable reference between resources.",
				Computed:    true,
			},
			"criteria": schema.ListNestedAttribute{
				Description: "The resolved criteria list. Identical to criteria_list but exported as an " +
					"attribute so it can be passed directly to other resources or combined with `concat()`.",
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"filter_entity": schema.StringAttribute{
							Computed: true,
						},
						"filter_field": schema.StringAttribute{
							Computed: true,
						},
						"filter_type": schema.StringAttribute{
							Computed: true,
						},
						"operator": schema.StringAttribute{
							Computed: true,
						},
						"value": schema.StringAttribute{
							Computed: true,
						},
						"case_sensitive": schema.BoolAttribute{
							Computed: true,
						},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"criteria_list": criteriaListBlock(),
		},
	}
}

// ModifyPlan resolves filter_type from filter_entity+filter_field shorthand so
// Terraform sees a known filter_type value before apply.
func (r *filterResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan filterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.CriteriaList.IsNull() || plan.CriteriaList.IsUnknown() {
		return
	}

	var blocks []criteriaListModel
	resp.Diagnostics.Append(plan.CriteriaList.ElementsAs(ctx, &blocks, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve shorthand → filter_type, same logic as group_resource.
	modified := false
	for i, block := range blocks {
		if !block.FilterType.IsUnknown() && !block.FilterType.IsNull() {
			continue
		}
		entity := block.FilterEntity.ValueString()
		field := block.FilterField.ValueString()
		if entity == "" || field == "" {
			continue
		}
		if fieldMap, ok := filterShorthandMap[entity]; ok {
			if ft, ok := fieldMap[field]; ok {
				blocks[i].FilterType = types.StringValue(ft)
				modified = true
			}
		}
	}

	// Rebuild criteria_list with resolved filter_type values.
	newList, d := types.ListValueFrom(ctx, criteriaObjectType(), blocks)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.CriteriaList = newList

	// Set criteria (computed output) equal to the resolved criteria_list.
	plan.Criteria = newList
	_ = modified

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

// Create stores the criteria in state without any API call.
func (r *filterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan filterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Assign a stable ID on first creation.
	plan.ID = types.StringValue(uuid.New().String())

	// Resolve and persist criteria.
	resolved, diags := resolveCriteriaToList(ctx, plan.CriteriaList)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Criteria = resolved

	tflog.Debug(ctx, "created turbonomic_filter (local)", map[string]interface{}{
		"id":    plan.ID.ValueString(),
		"count": len(plan.CriteriaList.Elements()),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read is a no-op - the resource has no external state to drift.
func (r *filterResource) Read(_ context.Context, _ resource.ReadRequest, _ *resource.ReadResponse) {}

// Update re-resolves criteria on change.
func (r *filterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan filterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve the existing ID.
	var state filterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID

	resolved, diags := resolveCriteriaToList(ctx, plan.CriteriaList)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Criteria = resolved

	tflog.Debug(ctx, "updated turbonomic_filter (local)", map[string]interface{}{
		"id":    plan.ID.ValueString(),
		"count": len(plan.CriteriaList.Elements()),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete is a no-op - there is no remote resource to destroy.
func (r *filterResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

// ImportState restores a filter from its ID. Because this resource is local-only,
// import is only meaningful if the state file was partially lost; the criteria_list
// will be empty and must be re-applied.
func (r *filterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// resolveCriteriaToList takes the criteria_list value from state/plan, resolves the
// filter_type shorthand for any block that needs it, and returns the result as a
// types.List of criteriaAttrTypes() objects. This is the value stored in `criteria`.
func resolveCriteriaToList(ctx context.Context, criteriaList types.List) (types.List, diag.Diagnostics) {
	emptyList := types.ListValueMust(criteriaObjectType(), []attr.Value{})

	if criteriaList.IsNull() || criteriaList.IsUnknown() {
		return emptyList, nil
	}

	var blocks []criteriaListModel
	diags := criteriaList.ElementsAs(ctx, &blocks, false)
	if diags.HasError() {
		return emptyList, diags
	}

	elements := make([]attr.Value, 0, len(blocks))
	for _, block := range blocks {
		// Resolve shorthand if filter_type is not yet set.
		if (block.FilterType.IsNull() || block.FilterType.IsUnknown() || block.FilterType.ValueString() == "") &&
			!block.FilterEntity.IsNull() && !block.FilterField.IsNull() {
			entity := block.FilterEntity.ValueString()
			field := block.FilterField.ValueString()
			if fieldMap, ok := filterShorthandMap[entity]; ok {
				if ft, ok := fieldMap[field]; ok {
					block.FilterType = types.StringValue(ft)
				}
			}
		}

		obj, err := buildCriteriaObject(block)
		if err != nil {
			diags.AddError("criteria object build error", err.Error())
			return emptyList, diags
		}
		elements = append(elements, obj)
	}

	result, d := types.ListValue(criteriaObjectType(), elements)
	diags.Append(d...)
	return result, diags
}
