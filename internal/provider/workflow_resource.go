// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	v2 "github.com/IBM/turbonomic-go-client/v2"
)

var (
	_ resource.Resource                = &workflowResource{}
	_ resource.ResourceWithConfigure   = &workflowResource{}
	_ resource.ResourceWithImportState = &workflowResource{}
)

// NewWorkflowResource returns a new turbonomic_workflow resource instance.
func NewWorkflowResource() resource.Resource {
	return &workflowResource{}
}

type workflowResource struct {
	v2Client *v2.Client
}

// workflowResourceModel is the Terraform state model for turbonomic_workflow.
type workflowResourceModel struct {
	ID               types.String `tfsdk:"id"`
	UUID             types.String `tfsdk:"uuid"`
	DisplayName      types.String `tfsdk:"display_name"`
	Description      types.String `tfsdk:"description"`
	Type             types.String `tfsdk:"type"`
	ActionType       types.String `tfsdk:"action_type"`
	ActionPhase      types.String `tfsdk:"action_phase"`
	EntityType       types.String `tfsdk:"entity_type"`
	TimeLimitSeconds types.Int64  `tfsdk:"time_limit_seconds"`
	// WEBHOOK fields
	WebhookURL             types.String `tfsdk:"webhook_url"`
	WebhookMethod          types.String `tfsdk:"webhook_method"`
	WebhookTemplate        types.String `tfsdk:"webhook_template"`
	WebhookAuthMethod      types.String `tfsdk:"webhook_auth_method"`
	WebhookUsername        types.String `tfsdk:"webhook_username"`
	WebhookPassword        types.String `tfsdk:"webhook_password"`
	WebhookTrustSelfSigned types.Bool   `tfsdk:"webhook_trust_self_signed_certificates"`
}

// workflowWriteDTO is the wire DTO for create/update (WorkflowApiDTO).
type workflowWriteDTO struct {
	DisplayName         string            `json:"displayName"`
	Description         *string           `json:"description,omitempty"`
	Type                string            `json:"type"`
	ActionType          *string           `json:"actionType,omitempty"`
	ActionPhase         *string           `json:"actionPhase,omitempty"`
	EntityType          *string           `json:"entityType,omitempty"`
	TimeLimitSeconds    *int64            `json:"timeLimitSeconds,omitempty"`
	TypeSpecificDetails *webhookAspectDTO `json:"typeSpecificDetails,omitempty"`
}

// webhookAspectDTO maps the WebhookApiDTO embedded in typeSpecificDetails.
type webhookAspectDTO struct {
	Type                 string  `json:"type"` // discriminator for WorkflowAspect - always "WEBHOOK"
	URL                  *string `json:"url,omitempty"`
	Method               *string `json:"method,omitempty"`
	Template             *string `json:"template,omitempty"`
	AuthenticationMethod *string `json:"authenticationMethod,omitempty"`
	Username             *string `json:"username,omitempty"`
	Password             *string `json:"password,omitempty"`
	TrustSelfSignedCerts *bool   `json:"trustSelfSignedCertificates,omitempty"`
}

func (r *workflowResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflow"
}

func (r *workflowResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Turbonomic workflow. " +
			"Workflows define external integrations (webhooks, action scripts, ServiceNow, Kafka) " +
			"that are triggered on action lifecycle events. " +
			"Currently supports WEBHOOK type; other orchestrator types can be referenced via the turbonomic_workflow data source.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Internal Terraform identifier, equal to the workflow UUID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uuid": schema.StringAttribute{
				Description: "UUID assigned by Turbonomic after creation.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"display_name": schema.StringAttribute{
				Description: "Human-readable name of the workflow.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Optional description of the workflow.",
				Optional:    true,
			},
			"type": schema.StringAttribute{
				Description: "Orchestrator type. Must be WEBHOOK for managed workflows. " +
					"Other types (ACTION_SCRIPT, SERVICENOW, ACTIONSTREAM_KAFKA, UCSD) are discovered automatically.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf("WEBHOOK", "ACTION_SCRIPT", "SERVICENOW", "ACTIONSTREAM_KAFKA", "UCSD"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"action_type": schema.StringAttribute{
				Description: "Action type the workflow applies to (e.g. RESIZE, MOVE, SUSPEND). Omit for all action types.",
				Optional:    true,
			},
			"action_phase": schema.StringAttribute{
				Description: "Action lifecycle phase: PRE, POST, or REPLACE.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("PRE", "POST", "REPLACE"),
				},
			},
			"entity_type": schema.StringAttribute{
				Description: "Entity type the workflow applies to (e.g. VirtualMachine). Omit for all entity types.",
				Optional:    true,
			},
			"time_limit_seconds": schema.Int64Attribute{
				Description: "Maximum execution time in seconds before the workflow is timed out. Default 600.",
				Optional:    true,
				Computed:    true,
			},
			"webhook_url": schema.StringAttribute{
				Description: "URL of the webhook endpoint. Required when type = WEBHOOK.",
				Optional:    true,
			},
			"webhook_method": schema.StringAttribute{
				Description: "HTTP method for the webhook call: GET, POST, PUT, DELETE, or PATCH. Default POST.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("GET", "POST", "PUT", "DELETE", "PATCH"),
				},
			},
			"webhook_template": schema.StringAttribute{
				Description: "JSON body template for the webhook request. Supports Turbonomic template variable substitution.",
				Optional:    true,
			},
			"webhook_auth_method": schema.StringAttribute{
				Description: "Authentication method for the webhook: NONE, BASIC, or OAUTH. Default NONE.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("NONE", "BASIC", "OAUTH"),
				},
			},
			"webhook_username": schema.StringAttribute{
				Description: "Username for BASIC authentication.",
				Optional:    true,
			},
			"webhook_password": schema.StringAttribute{ // pragma: allowlist secret
				Description: "Password for BASIC authentication. Write-only - never stored in state.",
				Optional:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"webhook_trust_self_signed_certificates": schema.BoolAttribute{
				Description: "Whether to trust self-signed TLS certificates on the webhook endpoint. Default false.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func (r *workflowResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
		resp.Diagnostics.AddError("v2 client not available", "The v2 client is required for workflow operations but is not available.")
	}
}

func (r *workflowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan workflowResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dto := buildWorkflowWriteDTO(plan)
	tflog.Debug(ctx, "creating workflow", map[string]interface{}{"display_name": plan.DisplayName.ValueString()})

	body, err := policyHTTPPost(ctx, r.v2Client.GetHTTPClient(), r.v2Client.GetBaseURL(), "/workflows", dto)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Workflow", err.Error())
		return
	}

	var created workflowDTO
	if err := json.Unmarshal(body, &created); err != nil {
		resp.Diagnostics.AddError("Error Parsing Create Response", err.Error())
		return
	}
	if created.UUID == nil {
		resp.Diagnostics.AddError("Error Creating Workflow", "API response did not include a UUID.")
		return
	}

	mapWorkflowDTOToModel(&created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *workflowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state workflowResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.UUID.ValueString()
	if uuid == "" {
		uuid = state.ID.ValueString()
	}

	body, statusCode, err := policyHTTPGet(ctx, r.v2Client.GetHTTPClient(), r.v2Client.GetBaseURL(), "/workflows/"+uuid)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Workflow", err.Error())
		return
	}
	if statusCode == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	var dto workflowDTO
	if err := json.Unmarshal(body, &dto); err != nil {
		resp.Diagnostics.AddError("Error Parsing Workflow Response", err.Error())
		return
	}

	mapWorkflowDTOToModel(&dto, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *workflowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan workflowResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state workflowResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.UUID = state.UUID
	plan.ID = state.ID
	uuid := state.UUID.ValueString()

	dto := buildWorkflowWriteDTO(plan)
	tflog.Debug(ctx, "updating workflow", map[string]interface{}{"uuid": uuid})

	body, err := policyHTTPPut(ctx, r.v2Client.GetHTTPClient(), r.v2Client.GetBaseURL(), "/workflows/"+uuid, dto)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Workflow", err.Error())
		return
	}

	var updated workflowDTO
	if err := json.Unmarshal(body, &updated); err != nil {
		resp.Diagnostics.AddError("Error Parsing Update Response", err.Error())
		return
	}

	mapWorkflowDTOToModel(&updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *workflowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state workflowResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	uuid := state.UUID.ValueString()
	tflog.Debug(ctx, "deleting workflow", map[string]interface{}{"uuid": uuid})
	if err := policyHTTPDelete(ctx, r.v2Client.GetHTTPClient(), r.v2Client.GetBaseURL(), "/workflows/"+uuid); err != nil {
		resp.Diagnostics.AddError("Error Deleting Workflow", err.Error())
	}
}

func (r *workflowResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildWorkflowWriteDTO(plan workflowResourceModel) workflowWriteDTO {
	dto := workflowWriteDTO{
		DisplayName: plan.DisplayName.ValueString(),
		Type:        plan.Type.ValueString(),
	}
	setIfKnown(&dto.Description, plan.Description, types.String.ValueString)
	setIfKnown(&dto.ActionType, plan.ActionType, types.String.ValueString)
	setIfKnown(&dto.ActionPhase, plan.ActionPhase, types.String.ValueString)
	setIfKnown(&dto.EntityType, plan.EntityType, types.String.ValueString)
	setIfKnown(&dto.TimeLimitSeconds, plan.TimeLimitSeconds, types.Int64.ValueInt64)

	// Build WebhookApiDTO for WEBHOOK type.
	// The API's typeSpecificDetails discriminator is "WebhookApiDTO", NOT "WEBHOOK".
	if plan.Type.ValueString() == "WEBHOOK" {
		aspect := &webhookAspectDTO{Type: "WebhookApiDTO"}
		setIfKnown(&aspect.URL, plan.WebhookURL, types.String.ValueString)
		setIfKnown(&aspect.Method, plan.WebhookMethod, types.String.ValueString)
		setIfKnown(&aspect.Template, plan.WebhookTemplate, types.String.ValueString)
		setIfKnown(&aspect.AuthenticationMethod, plan.WebhookAuthMethod, types.String.ValueString)
		setIfKnown(&aspect.Username, plan.WebhookUsername, types.String.ValueString)
		setIfKnown(&aspect.Password, plan.WebhookPassword, types.String.ValueString) // pragma: allowlist secret
		setIfKnown(&aspect.TrustSelfSignedCerts, plan.WebhookTrustSelfSigned, types.Bool.ValueBool)
		dto.TypeSpecificDetails = aspect
	}

	return dto
}

func mapWorkflowDTOToModel(dto *workflowDTO, model *workflowResourceModel) {
	if dto.UUID != nil {
		model.UUID = types.StringPointerValue(dto.UUID)
		model.ID = types.StringPointerValue(dto.UUID)
	}
	if dto.DisplayName != nil {
		model.DisplayName = types.StringPointerValue(dto.DisplayName)
	}
	if dto.Description != nil {
		model.Description = types.StringPointerValue(dto.Description)
	} else {
		model.Description = types.StringNull()
	}
	if dto.Type != nil {
		model.Type = types.StringPointerValue(dto.Type)
	}
	if dto.ActionType != nil {
		model.ActionType = types.StringPointerValue(dto.ActionType)
	} else {
		model.ActionType = types.StringNull()
	}
	if dto.ActionPhase != nil {
		model.ActionPhase = types.StringPointerValue(dto.ActionPhase)
	} else {
		model.ActionPhase = types.StringNull()
	}
	if dto.EntityType != nil {
		model.EntityType = types.StringPointerValue(dto.EntityType)
	} else {
		model.EntityType = types.StringNull()
	}
	// script_path is not in the resource schema (ACTION_SCRIPT only, not managed here)
	// but we still preserve it in the dto for the data source
	if dto.TimeLimitSeconds != nil {
		model.TimeLimitSeconds = types.Int64Value(*dto.TimeLimitSeconds)
	} else {
		model.TimeLimitSeconds = types.Int64Null()
	}
	// webhook_password is write-only - never overwrite from API response.

	// Populate webhook-specific Optional+Computed fields from typeSpecificDetails.
	// These must always be set (not unknown) after apply or Terraform raises an error.
	if dto.TypeSpecificDetails != nil {
		wh := dto.TypeSpecificDetails
		if wh.URL != nil {
			model.WebhookURL = types.StringPointerValue(wh.URL)
		}
		if wh.Method != nil {
			model.WebhookMethod = types.StringPointerValue(wh.Method)
		} else if model.WebhookMethod.IsUnknown() {
			model.WebhookMethod = types.StringNull()
		}
		if wh.Template != nil {
			model.WebhookTemplate = types.StringPointerValue(wh.Template)
		}
		if wh.AuthenticationMethod != nil {
			model.WebhookAuthMethod = types.StringPointerValue(wh.AuthenticationMethod)
		} else {
			// Default: API returns "NONE" when not explicitly set.
			model.WebhookAuthMethod = types.StringValue("NONE")
		}
		if wh.TrustSelfSignedCerts != nil {
			model.WebhookTrustSelfSigned = types.BoolPointerValue(wh.TrustSelfSignedCerts)
		} else {
			model.WebhookTrustSelfSigned = types.BoolValue(false)
		}
	} else {
		// No typeSpecificDetails returned - set defaults to keep state consistent.
		if model.WebhookAuthMethod.IsUnknown() {
			model.WebhookAuthMethod = types.StringNull()
		}
		if model.WebhookTrustSelfSigned.IsUnknown() {
			model.WebhookTrustSelfSigned = types.BoolValue(false)
		}
	}
}
