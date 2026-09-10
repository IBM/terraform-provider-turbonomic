// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	turboclient "github.com/IBM/turbonomic-go-client"
	v2 "github.com/IBM/turbonomic-go-client/v2"
)

var (
	_ resource.Resource                  = &groupResource{}
	_ resource.ResourceWithConfigure     = &groupResource{}
	_ resource.ResourceWithImportState   = &groupResource{}
	_ resource.ResourceWithModifyPlan    = &groupResource{}
)

func NewGroupResource() resource.Resource {
	return &groupResource{}
}

type groupResource struct {
	client   turboclient.T8cClient
	v2Client *v2.Client
}

type groupResourceModel struct {
	ID              types.String `tfsdk:"id"`
	DisplayName     types.String `tfsdk:"display_name"`
	IsStatic        types.Bool   `tfsdk:"is_static"`
	GroupType       types.String `tfsdk:"group_type"`
	EnvironmentType types.String `tfsdk:"environment_type"`
	CloudType       types.String `tfsdk:"cloud_type"`
	LogicalOperator types.String `tfsdk:"logical_operator"`
	MemberUuidList  types.List   `tfsdk:"member_uuid_list"`
	Scope           types.List   `tfsdk:"scope"`
	Temporary       types.Bool   `tfsdk:"temporary"`
	VendorIds       types.Map    `tfsdk:"vendor_ids"`
	CriteriaList    types.List   `tfsdk:"criteria_list"`

	// Computed fields
	UUID                types.String  `tfsdk:"uuid"`
	ClassName           types.String  `tfsdk:"class_name"`
	ActiveEntitiesCount types.Int64   `tfsdk:"active_entities_count"`
	EntitiesCount       types.Int64   `tfsdk:"entities_count"`
	MembersCount        types.Int64   `tfsdk:"members_count"`
	CostPrice           types.Float64 `tfsdk:"cost_price"`
	Severity            types.String  `tfsdk:"severity"`
	State               types.String  `tfsdk:"state"`
	EntityTypes         types.List    `tfsdk:"entity_types"`
	MemberTypes         types.List    `tfsdk:"member_types"`
}

type criteriaListModel struct {
	FilterEntity  types.String `tfsdk:"filter_entity"`
	FilterField   types.String `tfsdk:"filter_field"`
	FilterType    types.String `tfsdk:"filter_type"`
	Operator      types.String `tfsdk:"operator"`
	Value         types.String `tfsdk:"value"`
	CaseSensitive types.Bool   `tfsdk:"case_sensitive"`
}

// LinkDTO represents a relationship link
type LinkDTO struct {
	Rel map[string]interface{} `json:"rel,omitempty"`
}

// EntityReferenceDTO represents a basic entity reference
type EntityReferenceDTO struct {
	Links       []LinkDTO `json:"links,omitempty"`
	UUID        *string   `json:"uuid,omitempty"`
	DisplayName *string   `json:"displayName,omitempty"`
	ClassName   *string   `json:"className,omitempty"`
}

// StatisticsValuesDTO represents statistical values
type StatisticsValuesDTO struct {
	Max      *float64 `json:"max,omitempty"`
	Min      *float64 `json:"min,omitempty"`
	Avg      *float64 `json:"avg,omitempty"`
	Total    *float64 `json:"total,omitempty"`
	TotalMax *float64 `json:"totalMax,omitempty"`
	TotalMin *float64 `json:"totalMin,omitempty"`
}

// CommoditySourceDTO represents commodity source information
type CommoditySourceDTO struct {
	CapacityCommoditySource *string `json:"capacityCommoditySource,omitempty"`
	UsedCommoditySource     *string `json:"usedCommoditySource,omitempty"`
	CapacitySourceIndex     *int    `json:"capacitySourceIndex,omitempty"`
	UsedSourceIndex         *int    `json:"usedSourceIndex,omitempty"`
}

// HistUtilizationDTO represents historical utilization data
type HistUtilizationDTO struct {
	Type                              *string  `json:"type,omitempty"`
	Usage                             *float64 `json:"usage,omitempty"`
	Capacity                          *float64 `json:"capacity,omitempty"`
	ResizeMaxScalingObservationPeriod *int     `json:"resizeMaxScalingObservationPeriod,omitempty"`
	ResizeScalingAggressiveness       *int     `json:"resizeScalingAggressiveness,omitempty"`
}

// StatisticsFilterDTO represents a statistics filter
type StatisticsFilterDTO struct {
	Type        *string `json:"type,omitempty"`
	Value       *string `json:"value,omitempty"`
	DisplayName *string `json:"displayName,omitempty"`
}

// StatisticDTO represents a single statistic
type StatisticDTO struct {
	Links              []LinkDTO             `json:"links,omitempty"`
	UUID               *string               `json:"uuid,omitempty"`
	DisplayName        *string               `json:"displayName,omitempty"`
	ClassName          *string               `json:"className,omitempty"`
	Name               *string               `json:"name,omitempty"`
	Capacity           *StatisticsValuesDTO  `json:"capacity,omitempty"`
	Reserved           *StatisticsValuesDTO  `json:"reserved,omitempty"`
	RelatedEntityType  *string               `json:"relatedEntityType,omitempty"`
	Filters            []StatisticsFilterDTO `json:"filters,omitempty"`
	RelatedEntity      *EntityReferenceDTO   `json:"relatedEntity,omitempty"`
	NumRelatedEntities *int                  `json:"numRelatedEntities,omitempty"`
	Units              *string               `json:"units,omitempty"`
	Values             *StatisticsValuesDTO  `json:"values,omitempty"`
	Value              *float64              `json:"value,omitempty"`
	Predicted          *float64              `json:"predicted,omitempty"`
	CommoditySource    *CommoditySourceDTO   `json:"commoditySource,omitempty"`
	HistUtilizations   []HistUtilizationDTO  `json:"histUtilizations,omitempty"`
}

// StatsDTO represents statistics for a specific date
type StatsDTO struct {
	Links       []LinkDTO      `json:"links,omitempty"`
	UUID        *string        `json:"uuid,omitempty"`
	DisplayName *string        `json:"displayName,omitempty"`
	ClassName   *string        `json:"className,omitempty"`
	Date        *string        `json:"date,omitempty"`
	Statistics  []StatisticDTO `json:"statistics,omitempty"`
	Epoch       *string        `json:"epoch,omitempty"`
}

// InputFieldDTO represents an input field for a target
type InputFieldDTO struct {
	Links               []LinkDTO  `json:"links,omitempty"`
	UUID                *string    `json:"uuid,omitempty"`
	DisplayName         *string    `json:"displayName,omitempty"`
	ClassName           *string    `json:"className,omitempty"`
	Name                *string    `json:"name,omitempty"`
	Value               *string    `json:"value,omitempty"`
	DefaultValue        *string    `json:"defaultValue,omitempty"`
	IsMandatory         *bool      `json:"isMandatory,omitempty"`
	IsSecret            *bool      `json:"isSecret,omitempty"`
	IsMultiline         *bool      `json:"isMultiline,omitempty"`
	IsTargetDisplayName *bool      `json:"isTargetDisplayName,omitempty"`
	ValueType           *string    `json:"valueType,omitempty"`
	SpecificValueType   *string    `json:"specificValueType,omitempty"`
	Description         *string    `json:"description,omitempty"`
	VerificationRegex   *string    `json:"verificationRegex,omitempty"`
	GroupProperties     [][]string `json:"groupProperties,omitempty"`
	AllowedValues       []string   `json:"allowedValues,omitempty"`
	DependencyKey       *string    `json:"dependencyKey,omitempty"`
	DependencyValue     *string    `json:"dependencyValue,omitempty"`
	SecretManager       *struct {
		SecretProviderId *string `json:"secretProviderId,omitempty"`
		SecretPath       *string `json:"secretPath,omitempty"`
	} `json:"secretManager,omitempty"`
}

// PatchedFieldDTO represents a patched field
type PatchedFieldDTO struct {
	FieldName  *string `json:"fieldName,omitempty"`
	FieldValue *string `json:"fieldValue,omitempty"`
}

// PatchedTargetDTO represents a patched target
type PatchedTargetDTO struct {
	ProbeType     *string           `json:"probeType,omitempty"`
	PatchedFields []PatchedFieldDTO `json:"patchedFields,omitempty"`
}

// TargetErrorDetailsDTO represents target error details
type TargetErrorDetailsDTO struct {
	TargetErrorDetailsClass *string `json:"targetErrorDetailsClass,omitempty"`
}

// TargetHealthDTO represents target health information
type TargetHealthDTO struct {
	ErrorText                *string                 `json:"errorText,omitempty"`
	HealthCategory           *string                 `json:"healthCategory,omitempty"`
	HealthClassDiscriminator *string                 `json:"healthClassDiscriminator,omitempty"`
	UUID                     *string                 `json:"uuid,omitempty"`
	TargetName               *string                 `json:"targetName,omitempty"`
	RollupState              *string                 `json:"rollupState,omitempty"`
	TimeOfFirstFailure       *string                 `json:"timeOfFirstFailure,omitempty"`
	TargetErrorDetails       []TargetErrorDetailsDTO `json:"targetErrorDetails,omitempty"`
	TargetStatusSubcategory  *string                 `json:"targetStatusSubcategory,omitempty"`
	HealthState              *string                 `json:"healthState,omitempty"`
}

// OperationStageStatusDTO represents operation stage status
type OperationStageStatusDTO struct {
	State           *string `json:"state,omitempty"`
	Summary         *string `json:"summary,omitempty"`
	FullExplanation *string `json:"fullExplanation,omitempty"`
}

// OperationStageDTO represents an operation stage
type OperationStageDTO struct {
	Description *string                  `json:"description,omitempty"`
	Status      *OperationStageStatusDTO `json:"status,omitempty"`
	SubStages   []string                 `json:"subStages,omitempty"`
	Optional    *bool                    `json:"optional,omitempty"`
}

// HealthSummaryDTO represents health summary
type HealthSummaryDTO struct {
	HealthState                   *string `json:"healthState,omitempty"`
	RollupState                   *string `json:"rollupState,omitempty"`
	TimeOfLastSuccessfulDiscovery *string `json:"timeOfLastSuccessfulDiscovery,omitempty"`
}

// SourceDTO represents a target/source
type SourceDTO struct {
	Links                     []LinkDTO           `json:"links,omitempty"`
	UUID                      *string             `json:"uuid,omitempty"`
	DisplayName               *string             `json:"displayName,omitempty"`
	ClassName                 *string             `json:"className,omitempty"`
	Category                  *string             `json:"category,omitempty"`
	IsProbeRegistered         *bool               `json:"isProbeRegistered,omitempty"`
	UICategory                *string             `json:"uiCategory,omitempty"`
	IdentifyingFields         []string            `json:"identifyingFields,omitempty"`
	InputFields               []InputFieldDTO     `json:"inputFields,omitempty"`
	LastValidated             *string             `json:"lastValidated,omitempty"`
	Status                    *string             `json:"status,omitempty"`
	DerivedTargets            []string            `json:"derivedTargets,omitempty"`
	PatchedTargets            []PatchedTargetDTO  `json:"patchedTargets,omitempty"`
	ParentTargets             []string            `json:"parentTargets,omitempty"`
	Type                      *string             `json:"type,omitempty"`
	SubType                   *string             `json:"subType,omitempty"`
	Readonly                  *bool               `json:"readonly,omitempty"`
	Health                    *TargetHealthDTO    `json:"health,omitempty"`
	LastTargetOperationStages []OperationStageDTO `json:"lastTargetOperationStages,omitempty"`
	HealthSummary             *HealthSummaryDTO   `json:"healthSummary,omitempty"`
	DiscoveryMode             *string             `json:"discoveryMode,omitempty"`
}

// AspectDTO represents an aspect with type and SLI
type AspectDTO struct {
	Type *string  `json:"type,omitempty"`
	SLI  []string `json:"sli,omitempty"`
}

// FilterDTO represents a group filter criterion
type FilterDTO struct {
	CaseSensitive *bool   `json:"caseSensitive,omitempty"`
	EntityType    *string `json:"entityType,omitempty"`
	ExpType       string  `json:"expType"`
	ExpVal        *string `json:"expVal,omitempty"`
	FilterType    *string `json:"filterType,omitempty"`
	SingleLine    *bool   `json:"singleLine,omitempty"`
}

// operatorToExpType maps human-readable operator aliases to Turbonomic API exp_type values.
var operatorToExpType = map[string]string{
	"equals":              "EQ",
	"not_equals":          "NEQ",
	"greater_than":        "GT",
	"less_than":           "LT",
	"greater_than_equals": "GTE",
	"less_than_equals":    "LTE",
	"regex":               "RXEQ",
	"not_regex":           "RXNEQ",
	"exists":              "EX",
	"not_exists":          "NEX",
}

// expTypeToOperator is the reverse of operatorToExpType.
var expTypeToOperator = map[string]string{
	"EQ":    "equals",
	"NEQ":   "not_equals",
	"GT":    "greater_than",
	"LT":    "less_than",
	"GTE":   "greater_than_equals",
	"LTE":   "less_than_equals",
	"RXEQ":  "regex",
	"RXNEQ": "not_regex",
	"EX":    "exists",
	"NEX":   "not_exists",
}

// GroupDTO represents a Turbonomic group
type GroupDTO struct {
	UUID            *string           `json:"uuid,omitempty"`
	DisplayName     *string           `json:"displayName,omitempty"`
	IsStatic        *bool             `json:"isStatic,omitempty"`
	GroupType       *string           `json:"groupType,omitempty"`
	GroupClassName  *string           `json:"groupClassName,omitempty"`
	EnvironmentType *string           `json:"environmentType,omitempty"`
	CloudType       *string           `json:"cloudType,omitempty"`
	LogicalOperator *string           `json:"logicalOperator,omitempty"`
	MemberUuidList  *[]string         `json:"memberUuidList,omitempty"`
	Scope           *[]string         `json:"scope,omitempty"`
	GroupOrigin     *string           `json:"groupOrigin,omitempty"`
	TargetType      *string           `json:"targetType,omitempty"`
	Temporary       *bool             `json:"temporary,omitempty"`
	VendorIds       map[string]string `json:"vendorIds,omitempty"`
	CriteriaList    *[]FilterDTO      `json:"criteriaList,omitempty"`

	// Computed fields
	ClassName           *string   `json:"className,omitempty"`
	ActiveEntitiesCount *int64    `json:"activeEntitiesCount,omitempty"`
	EntitiesCount       *int64    `json:"entitiesCount,omitempty"`
	MembersCount        *int64    `json:"membersCount,omitempty"`
	CostPrice           *float64  `json:"costPrice,omitempty"`
	Severity            *string   `json:"severity,omitempty"`
	State               *string   `json:"state,omitempty"`
	EntityTypes         *[]string `json:"entityTypes,omitempty"`
	MemberTypes         *[]string `json:"memberTypes,omitempty"`

	// Additional computed fields from API
	Links                   []LinkDTO            `json:"links,omitempty"`
	RealtimeMarketReference *EntityReferenceDTO  `json:"realtimeMarketReference,omitempty"`
	Stats                   []StatsDTO           `json:"stats,omitempty"`
	Source                  *SourceDTO           `json:"source,omitempty"`
	Aspects                 map[string]AspectDTO `json:"aspects,omitempty"`
}

func (r *groupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (r *groupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Turbonomic group. Groups are static (fixed UUID list) or dynamic (criteria-based) collections of entities. Use them to scope placement policies, settings policies, and user access.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The UUID of the group (same as uuid).",
				Computed:    true,
			},
			"uuid": schema.StringAttribute{
				Description: "The UUID of the group.",
				Computed:    true,
			},
			"display_name": schema.StringAttribute{
				Description: "A user readable name of the group. Cannot be blank. The UTF-8 encoding must be at most 255 bytes.",
				Required:    true,
			},
			"is_static": schema.BoolAttribute{
				Description: "True if group is static (members defined by member_uuid_list), false if dynamic (members defined by criteria). Defaults to false.",
				Optional:    true,
			},
			"group_type": schema.StringAttribute{
				Description: "The type of service entities comprising the group.",
				Optional:    true,
			},
			"class_name": schema.StringAttribute{
				Description: "Class name of the group.",
				Computed:    true,
			},
			"environment_type": schema.StringAttribute{
				Description: "The environment type of the group. This is computed by the API based on the entities in the group and cannot be set directly.",
				Computed:    true,
			},
			"cloud_type": schema.StringAttribute{
				Description: "The cloud type of the group. This is computed by the API based on the entities in the group and cannot be set directly.",
				Computed:    true,
			},
			"logical_operator": schema.StringAttribute{
				Description: "Logical operator to be applied across all the criteria used to create dynamic group. Valid values: AND, OR, XOR.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("AND", "OR", "XOR"),
				},
			},
			"member_uuid_list": schema.ListAttribute{
				Description: "UUID list for members of the group - required if group is static.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"scope": schema.ListAttribute{
				Description: "Scope within which the criteria have to be applied to the list of specific entity type.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"temporary": schema.BoolAttribute{
				Description: "The Group is only valid for a limited period of time, it will be removed automatically.",
				Optional:    true,
			},
			"vendor_ids": schema.MapAttribute{
				Description: "The mapping of target identifier to vendor-provided identity of this group, if the group is discovered.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"active_entities_count": schema.Int64Attribute{
				Description: "The active entities count of a group.",
				Computed:    true,
			},
			"entities_count": schema.Int64Attribute{
				Description: "Number of entities of the Group.",
				Computed:    true,
			},
			"members_count": schema.Int64Attribute{
				Description: "Number of members of the Group.",
				Computed:    true,
			},
			"cost_price": schema.Float64Attribute{
				Description: "Cost of the Group per Hour: sum of the costs of the member entities.",
				Computed:    true,
			},
			"severity": schema.StringAttribute{
				Description: "Calculated using the highest severity of the member entities. Values: UNKNOWN, NORMAL, MINOR, MAJOR, CRITICAL.",
				Computed:    true,
			},
			"state": schema.StringAttribute{
				Description: "Calculated using the state of the member entities. Values: UNKNOWN, ACTIVE.",
				Computed:    true,
			},
			"entity_types": schema.ListAttribute{
				Description: "The types of entities contained in the group. This includes types of entities in nested levels of the group if the group is nested.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"member_types": schema.ListAttribute{
				Description: "The types for immediate members of the group.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
		Blocks: map[string]schema.Block{
			"criteria_list": criteriaListBlock(),
		},
	}
}

func (r *groupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = data.Client
	r.v2Client = data.V2Client

	if r.v2Client == nil {
		resp.Diagnostics.AddError(
			"v2 client not available",
			"The v2 client is required for group operations but is not available.",
		)
		return
	}
}

// validateGroupPlan enforces static/dynamic conditionality at plan time.
// It is extracted as a pure function so it can be unit-tested without constructing
// a live framework ModifyPlanRequest.
//
// Rules:
//   - is_static = true  → member_uuid_list required; criteria_list and logical_operator must not be set
//   - is_static = false → criteria_list required; member_uuid_list must not be set
func validateGroupPlan(plan groupResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	// is_static unknown during refresh/import - skip validation.
	if plan.IsStatic.IsNull() || plan.IsStatic.IsUnknown() {
		return diags
	}

	isStatic := plan.IsStatic.ValueBool()

	if isStatic {
		// Static group: member_uuid_list required.
		if plan.MemberUuidList.IsNull() || plan.MemberUuidList.IsUnknown() || len(plan.MemberUuidList.Elements()) == 0 {
			diags.AddAttributeError(
				path.Root("member_uuid_list"),
				"member_uuid_list required for static group",
				"Static groups (is_static = true) require at least one UUID in member_uuid_list.",
			)
		}
		// Static group: criteria_list must not be set.
		if !plan.CriteriaList.IsNull() && !plan.CriteriaList.IsUnknown() && len(plan.CriteriaList.Elements()) > 0 {
			diags.AddAttributeError(
				path.Root("criteria_list"),
				"criteria_list not allowed for static group",
				"criteria_list is only valid for dynamic groups (is_static = false). Remove it when is_static is true.",
			)
		}
		// Static group: logical_operator must not be set.
		if !plan.LogicalOperator.IsNull() && !plan.LogicalOperator.IsUnknown() {
			diags.AddAttributeError(
				path.Root("logical_operator"),
				"logical_operator not allowed for static group",
				"logical_operator is only valid for dynamic groups (is_static = false). Remove it when is_static is true.",
			)
		}
	} else {
		// Dynamic group: criteria_list required.
		if plan.CriteriaList.IsNull() || plan.CriteriaList.IsUnknown() || len(plan.CriteriaList.Elements()) == 0 {
			diags.AddAttributeError(
				path.Root("criteria_list"),
				"criteria_list required for dynamic group",
				"Dynamic groups (is_static = false or unset) require at least one criteria_list block.",
			)
		}
		// Dynamic group: member_uuid_list must not be set.
		if !plan.MemberUuidList.IsNull() && !plan.MemberUuidList.IsUnknown() && len(plan.MemberUuidList.Elements()) > 0 {
			diags.AddAttributeError(
				path.Root("member_uuid_list"),
				"member_uuid_list not allowed for dynamic group",
				"member_uuid_list is only valid for static groups (is_static = true). Remove it when is_static is false.",
			)
		}
	}

	return diags
}

// ModifyPlan resolves filter_type from filter_entity+filter_field at plan time so that
// Terraform sees a known value before apply, preventing "unknown value after apply" errors.
// It also validates static/dynamic conditionality via validateGroupPlan.
func (r *groupResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Only act on create/update plans (plan is null during destroy).
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan groupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate static/dynamic conditionality.
	resp.Diagnostics.Append(validateGroupPlan(plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.CriteriaList.IsNull() || plan.CriteriaList.IsUnknown() {
		return
	}

	var blocks []criteriaListModel
	diags = plan.CriteriaList.ElementsAs(ctx, &blocks, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	modified := false
	for i, block := range blocks {
		if !block.FilterType.IsUnknown() && !block.FilterType.IsNull() {
			// Already a known value - nothing to do.
			continue
		}
		// Attempt to resolve from shorthand.
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

	if !modified {
		return
	}

	newList, d := types.ListValueFrom(ctx, criteriaObjectType(), blocks)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.CriteriaList = newList

	diags = resp.Plan.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *groupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan groupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the group DTO
	groupDTO := GroupDTO{
		DisplayName: plan.DisplayName.ValueStringPointer(),
	}

	// Set optional input fields
	if !plan.IsStatic.IsNull() {
		isStatic := plan.IsStatic.ValueBool()
		groupDTO.IsStatic = &isStatic
	} else if !plan.CriteriaList.IsNull() && len(plan.CriteriaList.Elements()) > 0 {
		// criteria_list is set but is_static was not explicitly specified - this is a dynamic
		// group. Explicitly send isStatic=false so the API does not default to Static.
		isStatic := false
		groupDTO.IsStatic = &isStatic
	}

	if !plan.GroupType.IsNull() {
		groupDTO.GroupType = plan.GroupType.ValueStringPointer()
	}

	// Note: environment_type and cloud_type are NOT sent to the API
	// The API determines these based on the actual entities in the group

	if !plan.LogicalOperator.IsNull() {
		groupDTO.LogicalOperator = plan.LogicalOperator.ValueStringPointer()
	}

	// Always send GroupOrigin=USER - Terraform only manages user-created groups
	groupDTO.GroupOrigin = stringPtr("USER")

	if !plan.Temporary.IsNull() {
		temporary := plan.Temporary.ValueBool()
		groupDTO.Temporary = &temporary
	}

	// Handle member UUIDs
	if !plan.MemberUuidList.IsNull() {
		var memberUUIDs []string
		diags = plan.MemberUuidList.ElementsAs(ctx, &memberUUIDs, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		groupDTO.MemberUuidList = &memberUUIDs
	}

	// Handle scope
	if !plan.Scope.IsNull() {
		var scope []string
		diags = plan.Scope.ElementsAs(ctx, &scope, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		groupDTO.Scope = &scope
	}

	// Handle vendor IDs
	if !plan.VendorIds.IsNull() {
		var vendorIds map[string]string
		diags = plan.VendorIds.ElementsAs(ctx, &vendorIds, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		groupDTO.VendorIds = vendorIds
	}

	// Handle criteria list
	if !plan.CriteriaList.IsNull() && len(plan.CriteriaList.Elements()) > 0 {
		var criteriaBlocks []criteriaListModel
		diags = plan.CriteriaList.ElementsAs(ctx, &criteriaBlocks, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		criteriaList := make([]FilterDTO, 0, len(criteriaBlocks))
		for _, block := range criteriaBlocks {
			filterTypeStr, ok := resolveCriteriaFilterType(block, &resp.Diagnostics)
			if !ok {
				return
			}

			filter := FilterDTO{
				FilterType: &filterTypeStr,
				ExpType:    operatorToExpType[block.Operator.ValueString()],
			}

			if !block.Value.IsNull() {
				filter.ExpVal = block.Value.ValueStringPointer()
			}

			if !block.CaseSensitive.IsNull() {
				caseSensitive := block.CaseSensitive.ValueBool()
				filter.CaseSensitive = &caseSensitive
			}

			criteriaList = append(criteriaList, filter)
		}
		groupDTO.CriteriaList = &criteriaList
	}

	// Create the group via API using direct HTTP POST so that criteriaList and all fields
	// are serialized from GroupDTO without any intermediate conversion loss.
	tflog.Debug(ctx, "creating group", map[string]interface{}{
		"display_name": plan.DisplayName.ValueString(),
	})

	if r.v2Client == nil {
		resp.Diagnostics.AddError(
			"v2 client not available",
			"The v2 client is required for group operations but is not available.",
		)
		return
	}

	createdGroup, createErr := r.postGroupDTO(ctx, groupDTO)
	if createErr != nil {
		// If the group already exists (e.g. state was lost), adopt it and update it to match the plan.
		if alreadyExistsID := extractAlreadyExistsID(createErr.Error()); alreadyExistsID != "" {
			tflog.Debug(ctx, "group already exists, adopting existing group", map[string]interface{}{
				"display_name": plan.DisplayName.ValueString(),
				"existing_id":  alreadyExistsID,
			})
			groupDTO.UUID = &alreadyExistsID
			if _, updateErr := r.putGroupDTO(ctx, alreadyExistsID, groupDTO); updateErr != nil {
				resp.Diagnostics.AddError(
					"error creating group",
					fmt.Sprintf("Group already exists (id: %s) but could not be updated: %s", alreadyExistsID, updateErr.Error()),
				)
				return
			}
			// Re-fetch after adopt so all computed fields are fully resolved.
			var fetchErr error
			createdGroup, fetchErr = r.fetchGroupDTO(ctx, alreadyExistsID)
			if fetchErr != nil || createdGroup == nil {
				msg := alreadyExistsID + ": could not be re-read after adopt"
				if fetchErr != nil {
					msg = fetchErr.Error()
				}
				resp.Diagnostics.AddError("error creating group", fmt.Sprintf("Group already exists (id: %s) but could not be re-read: %s", alreadyExistsID, msg))
				return
			}
		} else {
			resp.Diagnostics.AddError(
				"error creating group",
				"Could not create group: "+createErr.Error(),
			)
			return
		}
	}

	// Re-fetch the created group so all computed fields (including CriteriaList.FilterType)
	// are fully resolved. The Create/adopt paths above may not return a complete response.
	if createdGroup == nil || createdGroup.UUID == nil {
		resp.Diagnostics.AddError(
			"error creating group",
			"API response did not include a UUID for the created group.",
		)
		return
	}
	fetchedGroup, fetchErr := r.fetchGroupDTO(ctx, *createdGroup.UUID)
	if fetchErr != nil {
		resp.Diagnostics.AddError(
			"error creating group",
			fmt.Sprintf("Group was created (id: %s) but could not be read back: %s", *createdGroup.UUID, fetchErr.Error()),
		)
		return
	}
	if fetchedGroup == nil {
		fetchedGroup = createdGroup
	}

	// Map response to state
	r.mapGroupDTOToModel(ctx, fetchedGroup, &plan, &resp.Diagnostics)

	// Save state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// fetchGroupDTO fetches a group by UUID via direct HTTP GET and decodes it into GroupDTO.
// Returns (nil, nil) when the group is not found (404).
func (r *groupResource) fetchGroupDTO(ctx context.Context, groupUUID string) (*GroupDTO, error) {
	url := fmt.Sprintf("%s/groups/%s?include_aspects=true", r.v2Client.GetBaseURL(), groupUUID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create HTTP request: %w", err)
	}

	httpResp, err := r.v2Client.GetHTTPClient().Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("could not fetch group %s: %w", groupUUID, err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", httpResp.StatusCode, string(body))
	}

	var group GroupDTO
	if err := json.NewDecoder(httpResp.Body).Decode(&group); err != nil {
		return nil, fmt.Errorf("could not decode group response: %w", err)
	}

	return &group, nil
}

// postGroupDTO creates a group via POST /groups using GroupDTO serialized directly as JSON.
func (r *groupResource) postGroupDTO(ctx context.Context, dto GroupDTO) (*GroupDTO, error) {
	jsonData, err := json.Marshal(dto)
	if err != nil {
		return nil, fmt.Errorf("could not marshal group: %w", err)
	}
	url := fmt.Sprintf("%s/groups", r.v2Client.GetBaseURL())
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("could not create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := r.v2Client.GetHTTPClient().Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("could not create group: %w", err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	body, _ := io.ReadAll(httpResp.Body)
	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("create group failed with status %d: %s", httpResp.StatusCode, string(body))
	}

	var created GroupDTO
	if err := json.Unmarshal(body, &created); err != nil {
		return nil, fmt.Errorf("could not decode create group response: %w", err)
	}
	return &created, nil
}

// putGroupDTO updates a group via PUT /groups/{uuid} using GroupDTO serialized directly as JSON.
func (r *groupResource) putGroupDTO(ctx context.Context, groupUUID string, dto GroupDTO) (*GroupDTO, error) {
	jsonData, err := json.Marshal(dto)
	if err != nil {
		return nil, fmt.Errorf("could not marshal group: %w", err)
	}
	url := fmt.Sprintf("%s/groups/%s", r.v2Client.GetBaseURL(), groupUUID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("could not create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := r.v2Client.GetHTTPClient().Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("could not update group %s: %w", groupUUID, err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	body, _ := io.ReadAll(httpResp.Body)
	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update group failed with status %d: %s", httpResp.StatusCode, string(body))
	}

	var updated GroupDTO
	if err := json.Unmarshal(body, &updated); err != nil {
		return nil, fmt.Errorf("could not decode update group response: %w", err)
	}
	return &updated, nil
}



func (r *groupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state groupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupUUID := state.ID.ValueString()
	tflog.Debug(ctx, "reading group", map[string]interface{}{
		"uuid": groupUUID,
	})

	if r.v2Client == nil {
		resp.Diagnostics.AddError(
			"v2 client not available",
			"The v2 client is required for group operations but is not available.",
		)
		return
	}

	group, err := r.fetchGroupDTO(ctx, groupUUID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Group",
			"Could not read group UUID "+groupUUID+": "+err.Error(),
		)
		return
	}
	if group == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Map response to state
	r.mapGroupDTOToModel(ctx, group, &state, &resp.Diagnostics)

	// Save state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *groupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan groupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state groupResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupUUID := state.ID.ValueString()

	// Build the group DTO
	groupDTO := GroupDTO{
		UUID:        &groupUUID,
		DisplayName: plan.DisplayName.ValueStringPointer(),
	}

	// Set optional fields
	if !plan.IsStatic.IsNull() {
		isStatic := plan.IsStatic.ValueBool()
		groupDTO.IsStatic = &isStatic
	} else if !plan.CriteriaList.IsNull() && len(plan.CriteriaList.Elements()) > 0 {
		// criteria_list is set but is_static was not explicitly specified - this is a dynamic
		// group. Explicitly send isStatic=false so the API does not default to Static.
		isStatic := false
		groupDTO.IsStatic = &isStatic
	}

	if !plan.GroupType.IsNull() {
		groupDTO.GroupType = plan.GroupType.ValueStringPointer()
	}

	// Note: environment_type and cloud_type are NOT sent to the API
	// The API determines these based on the actual entities in the group

	if !plan.LogicalOperator.IsNull() {
		groupDTO.LogicalOperator = plan.LogicalOperator.ValueStringPointer()
	}

	// Always send GroupOrigin=USER - Terraform only manages user-created groups
	groupDTO.GroupOrigin = stringPtr("USER")

	if !plan.Temporary.IsNull() {
		temporary := plan.Temporary.ValueBool()
		groupDTO.Temporary = &temporary
	}

	// Handle member UUIDs
	if !plan.MemberUuidList.IsNull() {
		var memberUUIDs []string
		diags = plan.MemberUuidList.ElementsAs(ctx, &memberUUIDs, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		groupDTO.MemberUuidList = &memberUUIDs
	}

	// Handle scope
	if !plan.Scope.IsNull() {
		var scope []string
		diags = plan.Scope.ElementsAs(ctx, &scope, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		groupDTO.Scope = &scope
	}

	// Handle vendor IDs
	if !plan.VendorIds.IsNull() {
		var vendorIds map[string]string
		diags = plan.VendorIds.ElementsAs(ctx, &vendorIds, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		groupDTO.VendorIds = vendorIds
	}

	// Handle criteria list
	if !plan.CriteriaList.IsNull() && len(plan.CriteriaList.Elements()) > 0 {
		var criteriaBlocks []criteriaListModel
		diags = plan.CriteriaList.ElementsAs(ctx, &criteriaBlocks, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		criteriaList := make([]FilterDTO, 0, len(criteriaBlocks))
		for _, block := range criteriaBlocks {
			filterTypeStr, ok := resolveCriteriaFilterType(block, &resp.Diagnostics)
			if !ok {
				return
			}

			filter := FilterDTO{
				FilterType: &filterTypeStr,
				ExpType:    operatorToExpType[block.Operator.ValueString()],
			}

			if !block.Value.IsNull() {
				filter.ExpVal = block.Value.ValueStringPointer()
			}

			if !block.CaseSensitive.IsNull() {
				caseSensitive := block.CaseSensitive.ValueBool()
				filter.CaseSensitive = &caseSensitive
			}

			criteriaList = append(criteriaList, filter)
		}
		groupDTO.CriteriaList = &criteriaList
	}

	// Update the group via API
	tflog.Debug(ctx, "updating group", map[string]interface{}{
		"uuid":         groupUUID,
		"display_name": plan.DisplayName.ValueString(),
	})

	if r.v2Client == nil {
		resp.Diagnostics.AddError(
			"v2 client not available",
			"The v2 client is required for group operations but is not available.",
		)
		return
	}

	updatedGroup, updateErr := r.putGroupDTO(ctx, groupUUID, groupDTO)
	if updateErr != nil {
		resp.Diagnostics.AddError(
			"error updating group",
			"Could not update group UUID "+groupUUID+": "+updateErr.Error(),
		)
		return
	}

	// Map response to state
	r.mapGroupDTOToModel(ctx, updatedGroup, &plan, &resp.Diagnostics)

	// Save state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *groupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state groupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupUUID := state.ID.ValueString()

	tflog.Debug(ctx, "deleting group", map[string]interface{}{
		"uuid": groupUUID,
	})

	if r.v2Client == nil {
		resp.Diagnostics.AddError(
			"v2 client not available",
			"The v2 client is required for group operations but is not available.",
		)
		return
	}

	url := fmt.Sprintf("%s/groups/%s", r.v2Client.GetBaseURL(), groupUUID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating HTTP Request",
			"Could not create HTTP request: "+err.Error(),
		)
		return
	}

	httpResp, err := r.v2Client.GetHTTPClient().Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Group",
			"Could not delete group UUID "+groupUUID+": "+err.Error(),
		)
		return
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode == http.StatusOK || httpResp.StatusCode == http.StatusNoContent || httpResp.StatusCode == http.StatusNotFound {
		return
	}
	body, _ := io.ReadAll(httpResp.Body)
	resp.Diagnostics.AddError(
		"Error Deleting Group",
		fmt.Sprintf("API returned status %d: %s", httpResp.StatusCode, string(body)),
	)
}

func (r *groupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helper function to map GroupDTO to model
func (r *groupResource) mapGroupDTOToModel(ctx context.Context, dto *GroupDTO, model *groupResourceModel, diags *diag.Diagnostics) {
	// Set ID and UUID
	if dto.UUID != nil {
		model.ID = types.StringPointerValue(dto.UUID)
		model.UUID = types.StringPointerValue(dto.UUID)
	}

	// Set input fields
	model.DisplayName = types.StringPointerValue(dto.DisplayName)

	// For optional fields, only update from API response if they were explicitly set in the plan.
	// This prevents API defaults from causing "inconsistent result after apply" errors.
	if !model.IsStatic.IsNull() {
		if dto.IsStatic != nil {
			model.IsStatic = types.BoolPointerValue(dto.IsStatic)
		}
	}
	// If is_static was null in the plan, keep it null regardless of what the API returns.

	if dto.GroupType != nil {
		model.GroupType = types.StringPointerValue(dto.GroupType)
	} else {
		model.GroupType = types.StringNull()
	}

	if !model.EnvironmentType.IsNull() {
		if dto.EnvironmentType != nil {
			// Accept any valid environment type from API (CLOUD, ONPREM, HYBRID)
			// The API determines this based on actual entities, not our input
			model.EnvironmentType = types.StringPointerValue(dto.EnvironmentType)
		}
	}
	// If it was null in plan, keep it null regardless of API response

	if !model.CloudType.IsNull() {
		if dto.CloudType != nil {
			// Accept any valid cloud type from API
			// The API determines this based on actual entities, not our input
			model.CloudType = types.StringPointerValue(dto.CloudType)
		}
	}
	// If it was null in plan, keep it null regardless of API response

	// For logical_operator: keep the plan value. The API may echo back a different default
	// (e.g. AND) even when OR was sent. We know what we intended, so preserve it.
	// If it was null in the plan, keep it null regardless of what the API returns.

	if !model.Temporary.IsNull() {
		if dto.Temporary != nil {
			model.Temporary = types.BoolPointerValue(dto.Temporary)
		}
	}
	// If it was null in plan, keep it null regardless of API response

	// Set computed fields
	if dto.ClassName != nil {
		model.ClassName = types.StringPointerValue(dto.ClassName)
	} else {
		model.ClassName = types.StringNull()
	}

	if dto.ActiveEntitiesCount != nil {
		model.ActiveEntitiesCount = types.Int64PointerValue(dto.ActiveEntitiesCount)
	} else {
		model.ActiveEntitiesCount = types.Int64Null()
	}

	if dto.EntitiesCount != nil {
		model.EntitiesCount = types.Int64PointerValue(dto.EntitiesCount)
	} else {
		model.EntitiesCount = types.Int64Null()
	}

	if dto.MembersCount != nil {
		model.MembersCount = types.Int64PointerValue(dto.MembersCount)
	} else {
		model.MembersCount = types.Int64Null()
	}

	if dto.CostPrice != nil {
		model.CostPrice = types.Float64PointerValue(dto.CostPrice)
	} else {
		model.CostPrice = types.Float64Null()
	}

	if dto.Severity != nil {
		model.Severity = types.StringPointerValue(dto.Severity)
	} else {
		model.Severity = types.StringNull()
	}

	if dto.State != nil {
		model.State = types.StringPointerValue(dto.State)
	} else {
		model.State = types.StringNull()
	}

	// Handle member UUIDs
	// For static groups: preserve plan values if API doesn't return them
	// For dynamic groups: keep null (API returns resolved members but we don't want them in state)
	isStatic := model.IsStatic.ValueBool()

	if isStatic {
		// Static group: update member list from API or preserve plan value
		if dto.MemberUuidList != nil && len(*dto.MemberUuidList) > 0 {
			memberElements := make([]attr.Value, len(*dto.MemberUuidList))
			for i, uuid := range *dto.MemberUuidList {
				memberElements[i] = types.StringValue(uuid)
			}
			var d diag.Diagnostics
			model.MemberUuidList, d = types.ListValue(types.StringType, memberElements)
			diags.Append(d...)
		} else if !model.MemberUuidList.IsNull() {
			// If API didn't return member list but we had one in plan, keep the plan value
		} else {
			model.MemberUuidList = types.ListNull(types.StringType)
		}
	} else {
		// Dynamic group: member_uuid_list should always be null
		// API returns resolved members but we don't store them in state
		model.MemberUuidList = types.ListNull(types.StringType)
	}

	// Handle scope - preserve plan values if API doesn't return them
	if dto.Scope != nil && len(*dto.Scope) > 0 {
		scopeElements := make([]attr.Value, len(*dto.Scope))
		for i, s := range *dto.Scope {
			scopeElements[i] = types.StringValue(s)
		}
		var d diag.Diagnostics
		model.Scope, d = types.ListValue(types.StringType, scopeElements)
		diags.Append(d...)
	} else if !model.Scope.IsNull() {
		// If API didn't return scope but we had one in plan, keep the plan value
	} else {
		model.Scope = types.ListNull(types.StringType)
	}

	// Handle entity types
	if dto.EntityTypes != nil && len(*dto.EntityTypes) > 0 {
		entityTypeElements := make([]attr.Value, len(*dto.EntityTypes))
		for i, et := range *dto.EntityTypes {
			entityTypeElements[i] = types.StringValue(et)
		}
		var d diag.Diagnostics
		model.EntityTypes, d = types.ListValue(types.StringType, entityTypeElements)
		diags.Append(d...)
	} else {
		model.EntityTypes = types.ListNull(types.StringType)
	}

	// Handle member types
	if dto.MemberTypes != nil && len(*dto.MemberTypes) > 0 {
		memberTypeElements := make([]attr.Value, len(*dto.MemberTypes))
		for i, mt := range *dto.MemberTypes {
			memberTypeElements[i] = types.StringValue(mt)
		}
		var d diag.Diagnostics
		model.MemberTypes, d = types.ListValue(types.StringType, memberTypeElements)
		diags.Append(d...)
	} else {
		model.MemberTypes = types.ListNull(types.StringType)
	}

	// Handle vendor IDs
	if len(dto.VendorIds) > 0 {
		vendorIdElements := make(map[string]attr.Value)
		for k, v := range dto.VendorIds {
			vendorIdElements[k] = types.StringValue(v)
		}
		var d diag.Diagnostics
		model.VendorIds, d = types.MapValue(types.StringType, vendorIdElements)
		diags.Append(d...)
	} else {
		model.VendorIds = types.MapNull(types.StringType)
	}

	// Handle criteria list - preserve plan values if API doesn't return them
	// Pre-decode the plan's criteria blocks so we can preserve user-set values that
	// the API may echo back with defaults (e.g. caseSensitive: false when user left it null).
	var planBlocks []criteriaListModel
	if !model.CriteriaList.IsNull() && !model.CriteriaList.IsUnknown() && len(model.CriteriaList.Elements()) > 0 {
		diags.Append(model.CriteriaList.ElementsAs(context.Background(), &planBlocks, false)...)
	}

	if dto.CriteriaList != nil && len(*dto.CriteriaList) > 0 {
		criteriaElements := make([]attr.Value, len(*dto.CriteriaList))
		for i, filter := range *dto.CriteriaList {
			// filter_type - always set from API response
			filterTypeVal := types.StringNull()
			if filter.FilterType != nil {
				filterTypeVal = types.StringPointerValue(filter.FilterType)
			}

			// filter_entity + filter_field - reverse-lookup from curated table.
			// Only populate them when the plan also had them set (i.e. the user used the
			// shorthand path). If the user used the raw filter_type escape hatch and left
			// filter_entity/filter_field null, keep them null here so Terraform does not
			// see a plan→state mismatch ("was null, but now cty.StringVal(…)").
			filterEntityVal := types.StringNull()
			filterFieldVal := types.StringNull()
			var planHadShorthand bool
			if i < len(planBlocks) {
				planHadShorthand = !planBlocks[i].FilterEntity.IsNull() || !planBlocks[i].FilterField.IsNull()
			}
			if planHadShorthand && filter.FilterType != nil {
				if pair, found := filterTypeToShorthand[*filter.FilterType]; found {
					filterEntityVal = types.StringValue(pair[0])
					filterFieldVal = types.StringValue(pair[1])
				}
			}

			// operator - reverse-map from API exp_type; fall back to raw value
			operatorVal := types.StringValue(filter.ExpType)
			if mapped, found := expTypeToOperator[filter.ExpType]; found {
				operatorVal = types.StringValue(mapped)
			}

			// value
			valueVal := types.StringNull()
			if filter.ExpVal != nil {
				valueVal = types.StringPointerValue(filter.ExpVal)
			}

			// case_sensitive - the API always returns false as a default even when the user
			// never set it. Preserve the plan's value: if plan had null, keep null; if plan
			// had an explicit bool, use the API's echo (which should match).
			var planCaseSensitive types.Bool
			if i < len(planBlocks) {
				planCaseSensitive = planBlocks[i].CaseSensitive
			}
			caseSensitive := types.BoolNull()
			if !planCaseSensitive.IsNull() && !planCaseSensitive.IsUnknown() {
				// User explicitly set case_sensitive - honour whatever the API echoes back.
				if filter.CaseSensitive != nil {
					caseSensitive = types.BoolPointerValue(filter.CaseSensitive)
				} else {
					caseSensitive = planCaseSensitive
				}
			}
			// planCaseSensitive was null/unknown → caseSensitive stays null.

			var d diag.Diagnostics
			obj, d := types.ObjectValue(criteriaAttrTypes(), map[string]attr.Value{
				"filter_entity":  filterEntityVal,
				"filter_field":   filterFieldVal,
				"filter_type":    filterTypeVal,
				"operator":       operatorVal,
				"value":          valueVal,
				"case_sensitive": caseSensitive,
			})
			diags.Append(d...)
			criteriaElements[i] = obj
		}

		var d diag.Diagnostics
		model.CriteriaList, d = types.ListValue(criteriaObjectType(), criteriaElements)
		diags.Append(d...)
	} else if !model.CriteriaList.IsNull() {
		// If API didn't return criteria list but we had one in plan, keep the plan value
		// This handles cases where API doesn't echo back the criteria list
	} else {
		model.CriteriaList = types.ListNull(criteriaObjectType())
	}
}

// extractAlreadyExistsID parses the numeric ID from a Turbonomic ALREADY_EXISTS error message.
// The API returns a message like:
//
//	ALREADY_EXISTS: Cannot create object with name X because an object with the same name (id: [287701365834688]) already exists.
//
// Returns the string ID if found, or an empty string.
func extractAlreadyExistsID(errMsg string) string {
	re := regexp.MustCompile(`ALREADY_EXISTS:.*\(id: \[(\d+)\]\)`)
	m := re.FindStringSubmatch(errMsg)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}
