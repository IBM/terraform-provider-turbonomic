// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	v2 "github.com/IBM/turbonomic-go-client/v2"
)

var _ datasource.DataSource = &workflowDataSource{}
var _ datasource.DataSourceWithConfigure = &workflowDataSource{}

// NewWorkflowDataSource returns a new instance of the turbonomic_workflow data source.
func NewWorkflowDataSource() datasource.DataSource {
	return &workflowDataSource{}
}

type workflowDataSource struct {
	v2Client *v2.Client
}

// workflowDataSourceModel is the root Terraform state model.
type workflowDataSourceModel struct {
	TypeFilter       types.String `tfsdk:"type"`
	DisplayNameFilter types.String `tfsdk:"display_name"`
	Workflows        types.List   `tfsdk:"workflows"`
}

// workflowDTO is the local wire DTO for a workflow returned by GET /workflows.
type workflowDTO struct {
	UUID                *string          `json:"uuid,omitempty"`
	DisplayName         *string          `json:"displayName,omitempty"`
	Description         *string          `json:"description,omitempty"`
	Type                *string          `json:"type,omitempty"`
	ActionType          *string          `json:"actionType,omitempty"`
	ActionPhase         *string          `json:"actionPhase,omitempty"`
	EntityType          *string          `json:"entityType,omitempty"`
	ScriptPath          *string          `json:"scriptPath,omitempty"`
	TimeLimitSeconds    *int64           `json:"timeLimitSeconds,omitempty"`
	TypeSpecificDetails *webhookReadDTO  `json:"typeSpecificDetails,omitempty"`
}

// webhookReadDTO is the read-side of typeSpecificDetails for WEBHOOK workflows.
type webhookReadDTO struct {
	URL                  *string `json:"url,omitempty"`
	Method               *string `json:"method,omitempty"`
	Template             *string `json:"template,omitempty"`
	AuthenticationMethod *string `json:"authenticationMethod,omitempty"`
	TrustSelfSignedCerts *bool   `json:"trustSelfSignedCertificates,omitempty"`
}

// workflowItemAttrTypes returns the canonical attr.Type map for a single workflow item.
func workflowItemAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"uuid":               types.StringType,
		"display_name":       types.StringType,
		"description":        types.StringType,
		"type":               types.StringType,
		"action_type":        types.StringType,
		"action_phase":       types.StringType,
		"entity_type":        types.StringType,
		"script_path":        types.StringType,
		"time_limit_seconds": types.Int64Type,
	}
}

func (d *workflowDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflow"
}

func (d *workflowDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	workflowItemSchema := schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the workflow.",
			},
			"display_name": schema.StringAttribute{
				Computed:    true,
				Description: "Human-readable name of the workflow.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Description of the workflow.",
			},
			"type": schema.StringAttribute{
				Computed:    true,
				Description: "Orchestrator type: WEBHOOK, ACTION_SCRIPT, SERVICENOW, ACTIONSTREAM_KAFKA, or UCSD.",
			},
			"action_type": schema.StringAttribute{
				Computed:    true,
				Description: "Action type the workflow is associated with (e.g. RESIZE, MOVE).",
			},
			"action_phase": schema.StringAttribute{
				Computed:    true,
				Description: "Action lifecycle phase the workflow applies to (e.g. PRE, POST, REPLACE).",
			},
			"entity_type": schema.StringAttribute{
				Computed:    true,
				Description: "Entity type the workflow is associated with (e.g. VirtualMachine).",
			},
			"script_path": schema.StringAttribute{
				Computed:    true,
				Description: "Full path to the workflow script. Set for ACTION_SCRIPT type.",
			},
			"time_limit_seconds": schema.Int64Attribute{
				Computed:    true,
				Description: "Execution time limit in seconds.",
			},
		},
	}

	resp.Schema = schema.Schema{
		Description: "Returns Turbonomic workflows, optionally filtered by type or display name. " +
			"Use this data source to look up workflow UUIDs for reference in other resources.",
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				Optional:    true,
				Description: "Filter results to workflows of this type (e.g. WEBHOOK, ACTION_SCRIPT). Omit to return all types.",
			},
			"display_name": schema.StringAttribute{
				Optional:    true,
				Description: "Filter results to the workflow whose display name matches exactly. Omit to return all workflows.",
			},
			"workflows": schema.ListNestedAttribute{
				Computed:     true,
				Description:  "List of workflows matching the supplied filters. Empty when no workflows match.",
				NestedObject: workflowItemSchema,
			},
		},
	}
}

func (d *workflowDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
			"The v2 client is required for workflow data source operations but is not available.",
		)
	}
}

func (d *workflowDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config workflowDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	typeFilter := config.TypeFilter.ValueString()
	displayNameFilter := config.DisplayNameFilter.ValueString()
	tflog.Debug(ctx, "reading workflow data source", map[string]interface{}{
		"type_filter":         typeFilter,
		"display_name_filter": displayNameFilter,
	})

	httpClient := d.v2Client.GetHTTPClient()
	baseURL := d.v2Client.GetBaseURL()
	url := baseURL + "/workflows"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		resp.Diagnostics.AddError("error building workflows request", err.Error())
		return
	}

	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("error fetching workflows", err.Error())
		return
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError(
			"unexpected status fetching workflows",
			fmt.Sprintf("GET %s returned HTTP %d", url, httpResp.StatusCode),
		)
		return
	}

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError("error reading workflows response body", err.Error())
		return
	}

	var allWorkflows []workflowDTO
	if err := json.Unmarshal(body, &allWorkflows); err != nil {
		resp.Diagnostics.AddError("error parsing workflows response", err.Error())
		return
	}

	itemType := types.ObjectType{AttrTypes: workflowItemAttrTypes()}
	var elements []attr.Value

	for i := range allWorkflows {
		w := &allWorkflows[i]
		if typeFilter != "" {
			if w.Type == nil || *w.Type != typeFilter {
				continue
			}
		}
		if displayNameFilter != "" {
			if w.DisplayName == nil || *w.DisplayName != displayNameFilter {
				continue
			}
		}
		obj, diags := buildWorkflowItemObject(w)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		elements = append(elements, obj)
	}

	if elements == nil {
		elements = []attr.Value{}
	}

	workflowsList, diags := types.ListValue(itemType, elements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "workflow data source read complete", map[string]interface{}{
		"total_fetched": len(allWorkflows),
		"matched":       len(elements),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &workflowDataSourceModel{
		TypeFilter:        config.TypeFilter,
		DisplayNameFilter: config.DisplayNameFilter,
		Workflows:         workflowsList,
	})...)
}

// buildWorkflowItemObject constructs a types.Object for a single workflowDTO.
func buildWorkflowItemObject(w *workflowDTO) (attr.Value, diag.Diagnostics) {
	uuid := types.StringNull()
	if w.UUID != nil {
		uuid = types.StringPointerValue(w.UUID)
	}
	displayName := types.StringNull()
	if w.DisplayName != nil {
		displayName = types.StringPointerValue(w.DisplayName)
	}
	description := types.StringNull()
	if w.Description != nil {
		description = types.StringPointerValue(w.Description)
	}
	wfType := types.StringNull()
	if w.Type != nil {
		wfType = types.StringPointerValue(w.Type)
	}
	actionType := types.StringNull()
	if w.ActionType != nil {
		actionType = types.StringPointerValue(w.ActionType)
	}
	actionPhase := types.StringNull()
	if w.ActionPhase != nil {
		actionPhase = types.StringPointerValue(w.ActionPhase)
	}
	entityType := types.StringNull()
	if w.EntityType != nil {
		entityType = types.StringPointerValue(w.EntityType)
	}
	scriptPath := types.StringNull()
	if w.ScriptPath != nil {
		scriptPath = types.StringPointerValue(w.ScriptPath)
	}
	timeLimitSeconds := types.Int64Null()
	if w.TimeLimitSeconds != nil {
		timeLimitSeconds = types.Int64Value(*w.TimeLimitSeconds)
	}

	return types.ObjectValue(workflowItemAttrTypes(), map[string]attr.Value{
		"uuid":               uuid,
		"display_name":       displayName,
		"description":        description,
		"type":               wfType,
		"action_type":        actionType,
		"action_phase":       actionPhase,
		"entity_type":        entityType,
		"script_path":        scriptPath,
		"time_limit_seconds": timeLimitSeconds,
	})
}
