// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/IBM/turbonomic-go-client/api/generated"
	v2 "github.com/IBM/turbonomic-go-client/v2"
)

var _ datasource.DataSource = &groupDataSource{}
var _ datasource.DataSourceWithConfigure = &groupDataSource{}

// NewGroupDataSource returns a new instance of the turbonomic_group data source.
func NewGroupDataSource() datasource.DataSource {
	return &groupDataSource{}
}

type groupDataSource struct {
	v2Client *v2.Client
}

// groupDataSourceModel is the root Terraform state model for the data source.
type groupDataSourceModel struct {
	DisplayNameFilter types.String `tfsdk:"display_name"`
	GroupTypeFilter   types.String `tfsdk:"group_type"`
	Groups            types.List   `tfsdk:"groups"`
}

// groupItemAttrTypes is the canonical attr.Type map for a single group item object.
// Used both when building object values and when constructing a null/empty list type.
func groupItemAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":                    types.StringType,
		"uuid":                  types.StringType,
		"display_name":          types.StringType,
		"group_type":            types.StringType,
		"is_static":             types.BoolType,
		"logical_operator":      types.StringType,
		"environment_type":      types.StringType,
		"cloud_type":            types.StringType,
		"class_name":            types.StringType,
		"active_entities_count": types.Int64Type,
		"entities_count":        types.Int64Type,
		"members_count":         types.Int64Type,
		"severity":              types.StringType,
		"state":                 types.StringType,
	}
}

func (d *groupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (d *groupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	groupItemSchema := schema.NestedAttributeObject{
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
				Description: "The human-readable name of the group.",
				Computed:    true,
			},
			"group_type": schema.StringAttribute{
				Description: "The entity type that makes up the group members (e.g. VirtualMachine, Storage).",
				Computed:    true,
			},
			"is_static": schema.BoolAttribute{
				Description: "True if the group has a fixed member list; false if membership is defined by criteria.",
				Computed:    true,
			},
			"logical_operator": schema.StringAttribute{
				Description: "Logical operator applied across all criteria for dynamic groups. Values: AND, OR, XOR.",
				Computed:    true,
			},
			"environment_type": schema.StringAttribute{
				Description: "The environment type of the group as determined by its members. Values: CLOUD, ONPREM, HYBRID.",
				Computed:    true,
			},
			"cloud_type": schema.StringAttribute{
				Description: "The cloud type of the group as determined by its members (e.g. AWS, AZURE, GCP).",
				Computed:    true,
			},
			"class_name": schema.StringAttribute{
				Description: "Internal API class name of the group object.",
				Computed:    true,
			},
			"active_entities_count": schema.Int64Attribute{
				Description: "Number of active entities in the group.",
				Computed:    true,
			},
			"entities_count": schema.Int64Attribute{
				Description: "Total number of entities in the group.",
				Computed:    true,
			},
			"members_count": schema.Int64Attribute{
				Description: "Number of direct members of the group.",
				Computed:    true,
			},
			"severity": schema.StringAttribute{
				Description: "Highest severity across member entities. Values: UNKNOWN, NORMAL, MINOR, MAJOR, CRITICAL.",
				Computed:    true,
			},
			"state": schema.StringAttribute{
				Description: "State derived from member entities. Values: UNKNOWN, ACTIVE.",
				Computed:    true,
			},
		},
	}

	resp.Schema = schema.Schema{
		Description: "Returns all Turbonomic groups, optionally filtered by display name (exact match) and/or group type. " +
			"Omitting both filters returns every group visible to the authenticated user.",
		Attributes: map[string]schema.Attribute{
			"display_name": schema.StringAttribute{
				Description: "Filter results to groups whose display name matches this value exactly. " +
					"Omit to return groups of any name.",
				Optional: true,
			},
			"group_type": schema.StringAttribute{
				Description: "Filter results to groups of this entity type (e.g. VirtualMachine, Storage). " +
					"Omit to return groups of any type.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"VirtualMachine", "PhysicalMachine", "Storage", "DiskArray",
						"Container", "ContainerPod", "ContainerSpec", "Namespace",
						"WorkloadController", "VirtualApplication",
						"Application", "ApplicationComponent",
						"BusinessTransaction", "Service",
						"Cluster", "StorageCluster", "VirtualMachineCluster",
						"Database", "DatabaseServer",
						"CloudVolume", "LoadBalancer",
						"BusinessAccount", "BillingFamily",
						"ResourceGroup", "Region", "Zone",
					),
				},
			},
			"groups": schema.ListNestedAttribute{
				Description:  "List of groups matching the supplied filters. Empty when no groups match.",
				Computed:     true,
				NestedObject: groupItemSchema,
			},
		},
	}
}

func (d *groupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
			"The v2 client is required for group data source operations but is not available.",
		)
	}
}

func (d *groupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config groupDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	displayNameFilter := config.DisplayNameFilter.ValueString()
	groupTypeFilter := config.GroupTypeFilter.ValueString()

	tflog.Debug(ctx, "reading group data source", map[string]interface{}{
		"display_name_filter": displayNameFilter,
		"group_type_filter":   groupTypeFilter,
	})

	// Paginate through all groups.
	var allGroups []generated.GroupApiDTO
	var cursor *string
	for {
		page, next, err := d.v2Client.Groups().List(ctx, cursor)
		if err != nil {
			resp.Diagnostics.AddError("error listing groups", err.Error())
			return
		}
		allGroups = append(allGroups, page...)
		if next == nil {
			break
		}
		cursor = next
	}

	// Apply filters and build result objects.
	itemType := types.ObjectType{AttrTypes: groupItemAttrTypes()}
	var matchedElements []attr.Value

	for i := range allGroups {
		g := &allGroups[i]

		if displayNameFilter != "" {
			if g.DisplayName == nil || *g.DisplayName != displayNameFilter {
				continue
			}
		}
		if groupTypeFilter != "" {
			if g.GroupType == nil || *g.GroupType != groupTypeFilter {
				continue
			}
		}

		obj, objDiags := buildGroupItemObject(g)
		resp.Diagnostics.Append(objDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		matchedElements = append(matchedElements, obj)
	}

	if matchedElements == nil {
		matchedElements = []attr.Value{}
	}

	groupsList, listDiags := types.ListValue(itemType, matchedElements)
	resp.Diagnostics.Append(listDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "group data source read complete", map[string]interface{}{
		"total_fetched": len(allGroups),
		"matched":       len(matchedElements),
	})

	state := groupDataSourceModel{
		DisplayNameFilter: config.DisplayNameFilter,
		GroupTypeFilter:   config.GroupTypeFilter,
		Groups:            groupsList,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// buildGroupItemObject constructs a types.Object for a single generated.GroupApiDTO.
func buildGroupItemObject(g *generated.GroupApiDTO) (attr.Value, diag.Diagnostics) {
	id := types.StringNull()
	uuid := types.StringNull()
	if g.Uuid != nil {
		id = types.StringPointerValue(g.Uuid)
		uuid = types.StringPointerValue(g.Uuid)
	}

	displayName := types.StringNull()
	if g.DisplayName != nil {
		displayName = types.StringPointerValue(g.DisplayName)
	}

	groupType := types.StringNull()
	if g.GroupType != nil {
		groupType = types.StringPointerValue(g.GroupType)
	}

	isStatic := types.BoolNull()
	if g.IsStatic != nil {
		isStatic = types.BoolPointerValue(g.IsStatic)
	}

	logicalOperator := types.StringNull()
	if g.LogicalOperator != nil {
		s := string(*g.LogicalOperator)
		logicalOperator = types.StringValue(s)
	}

	environmentType := types.StringNull()
	if g.EnvironmentType != nil {
		s := string(*g.EnvironmentType)
		environmentType = types.StringValue(s)
	}

	cloudType := types.StringNull()
	if g.CloudType != nil {
		s := string(*g.CloudType)
		cloudType = types.StringValue(s)
	}

	className := types.StringNull()
	if g.ClassName != nil {
		className = types.StringPointerValue(g.ClassName)
	}

	activeEntitiesCount := types.Int64Null()
	if g.ActiveEntitiesCount != nil {
		v := int64(*g.ActiveEntitiesCount)
		activeEntitiesCount = types.Int64Value(v)
	}

	entitiesCount := types.Int64Null()
	if g.EntitiesCount != nil {
		v := int64(*g.EntitiesCount)
		entitiesCount = types.Int64Value(v)
	}

	membersCount := types.Int64Null()
	if g.MembersCount != nil {
		v := int64(*g.MembersCount)
		membersCount = types.Int64Value(v)
	}

	severity := types.StringNull()
	if g.Severity != nil {
		severity = types.StringValue(string(*g.Severity))
	}

	state := types.StringNull()
	if g.State != nil {
		state = types.StringValue(string(*g.State))
	}

	return types.ObjectValue(groupItemAttrTypes(), map[string]attr.Value{
		"id":                    id,
		"uuid":                  uuid,
		"display_name":          displayName,
		"group_type":            groupType,
		"is_static":             isStatic,
		"logical_operator":      logicalOperator,
		"environment_type":      environmentType,
		"cloud_type":            cloudType,
		"class_name":            className,
		"active_entities_count": activeEntitiesCount,
		"entities_count":        entitiesCount,
		"members_count":         membersCount,
		"severity":              severity,
		"state":                 state,
	})
}
