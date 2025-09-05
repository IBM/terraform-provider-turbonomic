// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	turboclient "github.com/IBM/turbonomic-go-client"
)

var (
	_ datasource.DataSource              = &entityActionsDataSource{}
	_ datasource.DataSourceWithConfigure = &entityActionsDataSource{}
)

type EntityActionsModel struct {
	EntityUuid  types.String  `tfsdk:"entity_uuid"`
	EntityName  types.String  `tfsdk:"entity_name"`
	EntityType  types.String  `tfsdk:"entity_type"`
	ActionTypes types.List    `tfsdk:"action_types"`
	EnvType     types.String  `tfsdk:"environment_type"`
	States      types.List    `tfsdk:"states"`
	Actions     []ActionModel `tfsdk:"actions"`
}

type ActionModel struct {
	DisplayName types.String `tfsdk:"display_name"`
	ActionType  types.String `tfsdk:"action_type"`
	ActionState types.String `tfsdk:"action_state"`
	ActionMode  types.String `tfsdk:"action_mode"`
	Details     types.String `tfsdk:"details"`
	Target      struct {
		UUID            types.String `tfsdk:"uuid"`
		DisplayName     types.String `tfsdk:"display_name"`
		ClassName       types.String `tfsdk:"class_name"`
		EnvironmentType types.String `tfsdk:"environment_type"`
		DiscoveredBy    struct {
			UUID              types.String `tfsdk:"uuid"`
			DisplayName       types.String `tfsdk:"display_name"`
			IsProbeRegistered types.Bool   `tfsdk:"is_probe_registered"`
			Category          types.String `tfsdk:"category"`
			Type              types.String `tfsdk:"type"`
			Readonly          types.Bool   `tfsdk:"read_only"`
		} `tfsdk:"discovered_by"`
		VendorIds map[string]string    `tfsdk:"vendor_ids"`
		State     types.String         `tfsdk:"state"`
		Aspects   jsontypes.Normalized `tfsdk:"aspects"`
		Tags      map[string][]string  `tfsdk:"tags"`
	} `tfsdk:"target"`
	CurrentEntity struct {
		UUID            types.String `tfsdk:"uuid"`
		DisplayName     types.String `tfsdk:"display_name"`
		ClassName       types.String `tfsdk:"class_name"`
		EnvironmentType types.String `tfsdk:"environment_type"`
		DiscoveredBy    struct {
			UUID              types.String `tfsdk:"uuid"`
			DisplayName       types.String `tfsdk:"display_name"`
			IsProbeRegistered types.Bool   `tfsdk:"is_probe_registered"`
			Category          types.String `tfsdk:"category"`
			Type              types.String `tfsdk:"type"`
			Readonly          types.Bool   `tfsdk:"read_only"`
		} `tfsdk:"discovered_by"`
		VendorIds map[string]string `tfsdk:"vendor_ids"`
		State     types.String      `tfsdk:"state"`
	} `tfsdk:"current_entity"`
	NewEntity struct {
		UUID            types.String `tfsdk:"uuid"`
		DisplayName     types.String `tfsdk:"display_name"`
		ClassName       types.String `tfsdk:"class_name"`
		EnvironmentType types.String `tfsdk:"environment_type"`
		DiscoveredBy    struct {
			UUID              types.String `tfsdk:"uuid"`
			DisplayName       types.String `tfsdk:"display_name"`
			IsProbeRegistered types.Bool   `tfsdk:"is_probe_registered"`
			Category          types.String `tfsdk:"category"`
			Type              types.String `tfsdk:"type"`
			Readonly          types.Bool   `tfsdk:"read_only"`
		} `tfsdk:"discovered_by"`
		VendorIds map[string]string `tfsdk:"vendor_ids"`
		State     types.String      `tfsdk:"state"`
	} `tfsdk:"new_entity"`
	CurrentValue    types.String  `tfsdk:"current_value"`
	NewValue        types.String  `tfsdk:"new_value"`
	ValueUnits      types.String  `tfsdk:"value_units"`
	ResizeAttribute types.String  `tfsdk:"resize_attribute"`
	UUID            types.String  `tfsdk:"uuid"`
	ActionImpactID  types.Int64   `tfsdk:"action_impact_id"`
	MarketID        types.Int64   `tfsdk:"market_id"`
	CreateTime      types.String  `tfsdk:"create_time"`
	Importance      types.Float32 `tfsdk:"importance"`
	Template        struct {
		UUID        types.String `tfsdk:"uuid"`
		DisplayName types.String `tfsdk:"display_name"`
		ClassName   types.String `tfsdk:"class_name"`
		Discovered  types.Bool   `tfsdk:"discovered"`
		EnableMatch types.Bool   `tfsdk:"enable_match"`
	} `tfsdk:"template"`
	Risk struct {
		SubCategory       types.String   `tfsdk:"sub_category"`
		Description       types.String   `tfsdk:"description"`
		Severity          types.String   `tfsdk:"severity"`
		Importance        types.Float32  `tfsdk:"importance"`
		ReasonCommodities []types.String `tfsdk:"reason_commodities"`
	} `tfsdk:"risk"`
	Stats []struct {
		Name    types.String `tfsdk:"name"`
		Filters []struct {
			Type        types.String `tfsdk:"type"`
			Value       types.String `tfsdk:"value"`
			DisplayName types.String `tfsdk:"display_name"`
		} `tfsdk:"filters"`
		Units types.String  `tfsdk:"units"`
		Value types.Float64 `tfsdk:"value"`
	} `tfsdk:"stats"`
	CurrentLocation struct {
		UUID            types.String `tfsdk:"uuid"`
		DisplayName     types.String `tfsdk:"display_name"`
		ClassName       types.String `tfsdk:"class_name"`
		EnvironmentType types.String `tfsdk:"environment_type"`
		DiscoveredBy    struct {
			UUID              types.String `tfsdk:"uuid"`
			DisplayName       types.String `tfsdk:"display_name"`
			IsProbeRegistered types.Bool   `tfsdk:"is_probe_registered"`
			Category          types.String `tfsdk:"category"`
			Type              types.String `tfsdk:"type"`
			Readonly          types.Bool   `tfsdk:"read_only"`
		} `tfsdk:"discovered_by"`
		VendorIds map[string]string `tfsdk:"vendor_ids"`
	} `tfsdk:"current_location"`
	NewLocation struct {
		UUID            types.String `tfsdk:"uuid"`
		DisplayName     types.String `tfsdk:"display_name"`
		ClassName       types.String `tfsdk:"class_name"`
		EnvironmentType types.String `tfsdk:"environment_type"`
		DiscoveredBy    struct {
			UUID              types.String `tfsdk:"uuid"`
			DisplayName       types.String `tfsdk:"display_name"`
			IsProbeRegistered types.Bool   `tfsdk:"is_probe_registered"`
			Category          types.String `tfsdk:"category"`
			Type              types.String `tfsdk:"type"`
			Readonly          types.Bool   `tfsdk:"read_only"`
		} `tfsdk:"discovered_by"`
		VendorIds map[string]string `tfsdk:"vendor_ids"`
	} `tfsdk:"new_location"`
	CompoundActions []struct {
		DisplayName types.String `tfsdk:"display_name"`
		ActionType  types.String `tfsdk:"action_type"`
		ActionState types.String `tfsdk:"action_state"`
		ActionMode  types.String `tfsdk:"action_mode"`
		Details     types.String `tfsdk:"details"`
		Target      struct {
			UUID            types.String `tfsdk:"uuid"`
			DisplayName     types.String `tfsdk:"display_name"`
			ClassName       types.String `tfsdk:"class_name"`
			EnvironmentType types.String `tfsdk:"environment_type"`
			DiscoveredBy    struct {
				UUID              types.String `tfsdk:"uuid"`
				DisplayName       types.String `tfsdk:"display_name"`
				IsProbeRegistered types.Bool   `tfsdk:"is_probe_registered"`
				Category          types.String `tfsdk:"category"`
				Type              types.String `tfsdk:"type"`
				Readonly          types.Bool   `tfsdk:"read_only"`
			} `tfsdk:"discovered_by"`
			VendorIds map[string]string    `tfsdk:"vendor_ids"`
			State     types.String         `tfsdk:"state"`
			Aspects   jsontypes.Normalized `tfsdk:"aspects"`
			Tags      map[string][]string  `tfsdk:"tags"`
		} `tfsdk:"target"`
		CurrentEntity struct {
			UUID            types.String `tfsdk:"uuid"`
			DisplayName     types.String `tfsdk:"display_name"`
			ClassName       types.String `tfsdk:"class_name"`
			EnvironmentType types.String `tfsdk:"environment_type"`
			DiscoveredBy    struct {
				UUID              types.String `tfsdk:"uuid"`
				DisplayName       types.String `tfsdk:"display_name"`
				IsProbeRegistered types.Bool   `tfsdk:"is_probe_registered"`
				Category          types.String `tfsdk:"category"`
				Type              types.String `tfsdk:"type"`
				Readonly          types.Bool   `tfsdk:"read_only"`
			} `tfsdk:"discovered_by"`
			VendorIds map[string]string `tfsdk:"vendor_ids"`
			State     types.String      `tfsdk:"state"`
		} `tfsdk:"current_entity"`
		NewEntity struct {
			UUID            types.String `tfsdk:"uuid"`
			DisplayName     types.String `tfsdk:"display_name"`
			ClassName       types.String `tfsdk:"class_name"`
			EnvironmentType types.String `tfsdk:"environment_type"`
			DiscoveredBy    struct {
				UUID              types.String `tfsdk:"uuid"`
				DisplayName       types.String `tfsdk:"display_name"`
				IsProbeRegistered types.Bool   `tfsdk:"is_probe_registered"`
				Category          types.String `tfsdk:"category"`
				Type              types.String `tfsdk:"type"`
				Readonly          types.Bool   `tfsdk:"read_only"`
			} `tfsdk:"discovered_by"`
			VendorIds map[string]string `tfsdk:"vendor_ids"`
			State     types.String      `tfsdk:"state"`
		} `tfsdk:"new_entity"`
		CurrentValue    types.String `tfsdk:"current_value"`
		NewValue        types.String `tfsdk:"new_value"`
		ValueUnits      types.String `tfsdk:"value_units"`
		ResizeAttribute types.String `tfsdk:"resize_attribute"`
		Risk            struct {
			SubCategory       types.String   `tfsdk:"sub_category"`
			Description       types.String   `tfsdk:"description"`
			Severity          types.String   `tfsdk:"severity"`
			Importance        types.Float32  `tfsdk:"importance"`
			ReasonCommodities []types.String `tfsdk:"reason_commodities"`
		} `tfsdk:"risk"`
	} `tfsdk:"compound_actions"`
	Source   types.String `tfsdk:"source"`
	ActionID types.Int64  `tfsdk:"action_id"`
}

var (
	actionTypes      = []string{"START", "MOVE", "SCALE", "ALLOCATE", "SUSPEND", "PROVISION", "RECONFIGURE", "RESIZE", "DELETE", "RIGHT_SIZE", "BUY_RI"}
	environmentTypes = []string{"HYBRID", "CLOUD", "ONPREM", "UNKNOWN"}
	actionStates     = []string{"ACCEPTED", "REJECTED", "PRE_IN_PROGRESS", "POST_IN_PROGRESS", "IN_PROGRESS", "SUCCEEDED", "FAILED", "DISABLED", "QUEUED", "CLEARED", "ACCOUNTING", "READY", "FAILING", "BEFORE_EXEC", "IN_PROGRESS_NON_CRITICAL"}
)
var entityTypes = map[string]string{
	"applicationcomponentspec": "ApplicationComponentSpec",
	"applicationcomponent":     "ApplicationComponent",
	"availabilityzone":         "AvailabilityZone",
	"billingfamily":            "BillingFamily",
	"businessaccountfolder":    "BusinessAccountFolder",
	"businessaccount":          "BusinessAccount",
	"businessapplication":      "BusinessApplication",
	"businesstransaction":      "BusinessTransaction",
	"businessuser":             "BusinessUser",
	"chassis":                  "Chassis",
	"cluster":                  "Cluster",
	"computetier":              "ComputeTier",
	"containerplatformcluster": "ContainerPlatformCluster",
	"container":                "Container",
	"containerpod":             "ContainerPod",
	"containerspec":            "ContainerSpec",
	"datacenter":               "DataCenter",
	"database":                 "Database",
	"databaseserver":           "DatabaseServer",
	"databaseservertier":       "DatabaseServerTier",
	"databasetier":             "DatabaseTier",
	"desktoppool":              "DesktopPool",
	"diskarray":                "DiskArray",
	"documentcollection":       "DocumentCollection",
	"group":                    "Group",
	"iomodule":                 "IOModule",
	"internet":                 "Internet",
	"loadbalancer":             "LoadBalancer",
	"logicalpool":              "LogicalPool",
	"namespace":                "Namespace",
	"network":                  "Network",
	"physicalmachine":          "PhysicalMachine",
	"region":                   "Region",
	"resourcegroup":            "ResourceGroup",
	"service":                  "Service",
	"storage":                  "Storage",
	"storagecluster":           "StorageCluster",
	"storagecontroller":        "StorageController",
	"storagetier":              "StorageTier",
	"switch":                   "Switch",
	"viewpod":                  "ViewPod",
	"virtualdatacenter":        "VirtualDataCenter",
	"virtualmachine":           "VirtualMachine",
	"virtualmachinecluster":    "VirtualMachineCluster",
	"virtualmachinespec":       "VirtualMachineSpec",
	"virtualvolume":            "VirtualVolume",
	"workload":                 "Workload",
	"workloadcontroller":       "WorkloadController",
}

func NewEntityActionsDataSource() datasource.DataSource {
	return &entityActionsDataSource{}
}

type entityActionsDataSource struct {
	client *turboclient.Client
}

func (d *entityActionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_entity_actions"
}

func (d *entityActionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The following example demonstrates the syntax for the `turbonomic_entity_actions` data source. This can be used to access the DTO of actions",
		Attributes: map[string]schema.Attribute{
			"entity_uuid": schema.StringAttribute{
				MarkdownDescription: "Turbonomic UUID of the entity",
				Computed:            true,
			},
			"entity_name": schema.StringAttribute{
				MarkdownDescription: "case sensitive name of the entity",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"entity_type": schema.StringAttribute{
				MarkdownDescription: "case insensitive type of the entity",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive(slices.Collect(maps.Values(entityTypes))...),
				},
			},
			"action_types": schema.ListAttribute{
				MarkdownDescription: "type of the action",
				ElementType:         types.StringType,
				Optional:            true,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOfCaseInsensitive(actionTypes...)),
				},
			},
			"environment_type": schema.StringAttribute{
				MarkdownDescription: "filter the actions by environment type",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive(environmentTypes...),
				},
			},
			"states": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "list of states to filter",
				Optional:            true,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOfCaseInsensitive(actionStates...)),
				},
			},

			"actions": schema.ListNestedAttribute{
				MarkdownDescription: "list of actions",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							MarkdownDescription: "uuid of the action",
							Computed:            true,
						},
						"display_name": schema.StringAttribute{
							MarkdownDescription: "a user readable name of the api object",
							Computed:            true,
						},
						"action_impact_id": schema.Int64Attribute{
							MarkdownDescription: "the ID for the action, which will persist across restarts",
							Computed:            true,
						},
						"market_id": schema.Int64Attribute{
							MarkdownDescription: "the ID of the market for which the action was generated",
							Computed:            true,
						},
						"create_time": schema.StringAttribute{
							MarkdownDescription: "creation time",
							Computed:            true,
						},
						"action_type": schema.StringAttribute{
							MarkdownDescription: "type of the action",
							Computed:            true,
						},
						"action_state": schema.StringAttribute{
							MarkdownDescription: "state of the action",
							Computed:            true,
						},
						"action_mode": schema.StringAttribute{
							MarkdownDescription: "action mode",
							Computed:            true,
						},
						"details": schema.StringAttribute{
							MarkdownDescription: "a user-readable string describing the action",
							Computed:            true,
						},
						"importance": schema.Float32Attribute{
							MarkdownDescription: "numeric value that describes the priority of the action",
							Computed:            true,
						},
						"target": schema.SingleNestedAttribute{
							MarkdownDescription: "target entity for an action. For example, the VM in a Resize Action, or the host for a VM move",
							Computed:            true,
							Attributes:          insertEntitySchema(true, true, true),
						},
						"current_entity": schema.SingleNestedAttribute{
							MarkdownDescription: "current entity, such as the current host that a VM resides on for a VM move action",
							Computed:            true,
							Attributes:          insertEntitySchema(true, false, false),
						},
						"new_entity": schema.SingleNestedAttribute{
							MarkdownDescription: "destination entity, such as the host that a VM will move to for a VM move action",
							Computed:            true,
							Attributes:          insertEntitySchema(true, false, false),
						},
						"current_value": schema.StringAttribute{
							MarkdownDescription: "current value of a property, for example vMEM for a VM resize action",
							Computed:            true,
						},
						"new_value": schema.StringAttribute{
							MarkdownDescription: "calculated value to resize to, such as vMEM for a VM resize action",
							Computed:            true,
						},
						"value_units": schema.StringAttribute{
							MarkdownDescription: "units of the currentValue and newValue, such as KB for a VM vMEM resize action",
							Computed:            true,
						},
						"resize_attribute": schema.StringAttribute{
							MarkdownDescription: "the commodity attribute to be resized",
							Computed:            true,
						},
						"template": schema.SingleNestedAttribute{
							MarkdownDescription: "template used for the action, such as in a Cloud entity provision or move action",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"uuid": schema.StringAttribute{
									MarkdownDescription: "uuid of the action",
									Computed:            true,
								},
								"display_name": schema.StringAttribute{
									MarkdownDescription: "a user readable name of the api object",
									Computed:            true,
								},
								"class_name": schema.StringAttribute{
									MarkdownDescription: "class of the entity",
									Computed:            true,
								},
								"discovered": schema.BoolAttribute{
									MarkdownDescription: "indicates if the template is discovered or manually created",
									Computed:            true,
								},
								"enable_match": schema.BoolAttribute{
									MarkdownDescription: "add to infrastructure cost policy, infrastructure cost policies group hardware devices according to their cost",
									Computed:            true,
								},
							},
						},
						"risk": schema.SingleNestedAttribute{
							MarkdownDescription: "risk information of an entity",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"sub_category": schema.StringAttribute{
									MarkdownDescription: "subcategory of risk",
									Computed:            true,
								},
								"description": schema.StringAttribute{
									MarkdownDescription: "description of risk",
									Computed:            true,
								},
								"severity": schema.StringAttribute{
									MarkdownDescription: "severity of risk",
									Computed:            true,
								},
								"importance": schema.Float32Attribute{
									MarkdownDescription: "numeric value that describes the priority of the risk",
									Computed:            true,
								},
								"reason_commodities": schema.ListAttribute{
									MarkdownDescription: "the distinct set of commodities that were the reason for the action; not all actions are driven by commodities, so its possible that this can be an empty list.",
									Computed:            true,
									ElementType:         types.StringType,
								},
							},
						},
						"stats": schema.ListNestedAttribute{
							MarkdownDescription: "statistics, such as Mem, vCPU, costPrice",
							Computed:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										MarkdownDescription: "name of statistic",
										Computed:            true,
									},
									"filters": schema.ListNestedAttribute{
										MarkdownDescription: "describe the grouping options used to generate the output",
										Computed:            true,
										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{

												"type": schema.StringAttribute{
													MarkdownDescription: "type of the filter, E.G: action_types, category, ...",
													Computed:            true,
												},
												"value": schema.StringAttribute{
													MarkdownDescription: "value of the filter",
													Computed:            true,
												},
												"display_name": schema.StringAttribute{
													MarkdownDescription: "display name of the value, E.G: display_name if 'value' is an oid or an enum",
													Computed:            true,
												},
											},
										},
									},
									"units": schema.StringAttribute{
										MarkdownDescription: "units, used for Commodities stats. E.G. $/h",
										Computed:            true,
									},
									"value": schema.Float64Attribute{
										MarkdownDescription: "simple value, equal to values.avg",
										Computed:            true,
									},
								},
							},
						},
						"current_location": schema.SingleNestedAttribute{
							MarkdownDescription: "the region (DataCenter) where the current service entity is located, for cloud migration actions",
							Computed:            true,
							Attributes:          insertEntitySchema(false, false, false),
						},
						"new_location": schema.SingleNestedAttribute{
							MarkdownDescription: "the region, represented as a DataCenter entity, where the target service entity will be located for cloud migration actions",
							Computed:            true,
							Attributes:          insertEntitySchema(false, false, false),
						},
						"compound_actions": schema.ListNestedAttribute{
							MarkdownDescription: "property for compound actions",
							Computed:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"display_name": schema.StringAttribute{
										MarkdownDescription: "a user readable name of the api object",
										Computed:            true,
									},
									"action_type": schema.StringAttribute{
										MarkdownDescription: "type of the action",
										Computed:            true,
									},
									"action_state": schema.StringAttribute{
										MarkdownDescription: "state of the action",
										Computed:            true,
									},
									"action_mode": schema.StringAttribute{
										MarkdownDescription: "action mode",
										Computed:            true,
									},
									"details": schema.StringAttribute{
										MarkdownDescription: "a user-readable string describing the action",
										Computed:            true,
									},
									"target": schema.SingleNestedAttribute{
										MarkdownDescription: "target entity for an action. For example, the VM in a Resize Action, or the host for a VM move",
										Computed:            true,
										Attributes:          insertEntitySchema(true, true, true),
									},
									"current_entity": schema.SingleNestedAttribute{
										MarkdownDescription: "current entity, such as the current host that a VM resides on for a VM move action",
										Computed:            true,
										Attributes:          insertEntitySchema(true, false, false),
									},
									"new_entity": schema.SingleNestedAttribute{
										MarkdownDescription: "destination entity, such as the host that a VM will move to for a VM move action",
										Computed:            true,
										Attributes:          insertEntitySchema(true, false, false),
									},
									"current_value": schema.StringAttribute{
										MarkdownDescription: "current value of a property, for example vMEM for a VM resize action",
										Computed:            true,
									},
									"new_value": schema.StringAttribute{
										MarkdownDescription: "calculated value to resize to, such as vMEM for a VM resize action",
										Computed:            true,
									},
									"value_units": schema.StringAttribute{
										MarkdownDescription: "units of the currentValue and newValue, such as KB for a VM vMEM resize action",
										Computed:            true,
									},
									"resize_attribute": schema.StringAttribute{
										MarkdownDescription: "the commodity attribute to be resized",
										Computed:            true,
									},
									"risk": schema.SingleNestedAttribute{
										MarkdownDescription: "risk information of an entity",
										Computed:            true,
										Attributes: map[string]schema.Attribute{
											"sub_category": schema.StringAttribute{
												MarkdownDescription: "subcategory of risk",
												Computed:            true,
											},
											"description": schema.StringAttribute{
												MarkdownDescription: "description of risk",
												Computed:            true,
											},
											"severity": schema.StringAttribute{
												MarkdownDescription: "severity of risk",
												Computed:            true,
											},
											"importance": schema.Float32Attribute{
												MarkdownDescription: "numeric value that describes the priority of the risk",
												Computed:            true,
											},
											"reason_commodities": schema.ListAttribute{
												MarkdownDescription: "the distinct set of commodities that were the reason for the action; not all actions are driven by commodities, so its possible that this can be an empty list.",
												Computed:            true,
												ElementType:         types.StringType,
											},
										},
									},
								},
							},
						},
						"source": schema.StringAttribute{
							MarkdownDescription: "defines what generated this action",
							Computed:            true,
						},
						"action_id": schema.Int64Attribute{
							MarkdownDescription: "id of the action",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func insertEntitySchema(includeState bool, includeAspects bool, includeTags bool) map[string]schema.Attribute {
	attribs := map[string]schema.Attribute{
		"uuid": schema.StringAttribute{
			MarkdownDescription: "uuid of the target entity",
			Computed:            true,
		},
		"display_name": schema.StringAttribute{
			MarkdownDescription: "a user readable name of the api object",
			Computed:            true,
		},
		"class_name": schema.StringAttribute{
			MarkdownDescription: "a user readable name of the api object",
			Computed:            true,
		},
		"environment_type": schema.StringAttribute{
			MarkdownDescription: "environment type",
			Computed:            true,
		},
		"discovered_by": schema.SingleNestedAttribute{
			MarkdownDescription: "target that discovered the entity",
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"uuid": schema.StringAttribute{
					MarkdownDescription: "uuid of the discoveryBy target",
					Computed:            true,
				},
				"display_name": schema.StringAttribute{
					MarkdownDescription: "a user readable name of the api object",
					Computed:            true,
				},
				"category": schema.StringAttribute{
					MarkdownDescription: "probe category",
					Computed:            true,
				},
				"is_probe_registered": schema.BoolAttribute{
					MarkdownDescription: "indicator that is used to determine whether the associated probe is running and registered with the system",
					Computed:            true,
				},
				"type": schema.StringAttribute{
					MarkdownDescription: "probe type",
					Computed:            true,
				},
				"read_only": schema.BoolAttribute{
					MarkdownDescription: "whether the target cannot be changed through public APIs",
					Computed:            true,
				},
			},
		},
		"vendor_ids": schema.MapAttribute{
			MarkdownDescription: "the mapping of target identifier to vendor-provided identity of this entity on the remote target",
			Computed:            true,
			ElementType:         types.StringType,
		},
	}

	if includeState {
		attribs["state"] = schema.StringAttribute{
			MarkdownDescription: "state",
			Computed:            true,
		}
	}

	if includeAspects {
		attribs["aspects"] = schema.StringAttribute{
			MarkdownDescription: "additional info about the Entity categorized as Aspects",
			CustomType:          jsontypes.NormalizedType{},
			Computed:            true,
		}
	}

	if includeTags {
		attribs["tags"] = schema.MapAttribute{
			Description: "tags are the metadata defined in name/value pairs. Each name can have multiple values.",
			ElementType: types.ListType{ElemType: types.StringType},
			Computed:    true,
		}
	}

	return attribs
}

func (d *entityActionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*turboclient.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected: *turboclient.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *entityActionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state EntityActionsModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	enName, enTyp, envType := state.EntityName.ValueString(), state.EntityType.ValueString(), strings.ToUpper(state.EnvType.ValueString())
	var actTypes, actStates []string
	err := state.ActionTypes.ElementsAs(ctx, &actTypes, true)
	if err.HasError() {
		resp.Diagnostics.Append(err.Errors()...)
		return
	}
	err = state.States.ElementsAs(ctx, &actStates, true)
	if err.HasError() {
		resp.Diagnostics.Append(err.Errors()...)
		return
	}

	entity, errDiag := GetEntitiesByNameAndType(d.client, enName, entityTypes[strings.ToLower(enTyp)], envType, "")
	if errDiag != nil {
		tflog.Error(ctx, errDiag.Detail())
		resp.Diagnostics.AddError(errDiag.Summary(), errDiag.Detail())
		return
	} else if len(entity) == 0 {
		errDetail := fmt.Sprintf("entity %s of type %s not found in Turbonomic instance", enName, enTyp)
		tflog.Warn(ctx, errDetail)

		state.EntityType = types.StringNull()
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("entity id found: %s\n", entity[0].UUID))
	state.EntityUuid = types.StringValue(entity[0].UUID)

	actions, errDiag := GetFilteredEntityActions(d.client, entity[0].UUID, actTypes, actStates)
	if errDiag != nil {
		tflog.Error(ctx, errDiag.Detail())
		resp.Diagnostics.AddError(errDiag.Summary(), errDiag.Detail())
		return
	} else if len(actions) == 0 {
		errDetail := fmt.Sprintf("no matching action found for entity id: %s", entity[0].UUID)
		tflog.Trace(ctx, errDetail)

		if err := TagEntity(d.client, state.EntityUuid.ValueString()); err != nil {
			resp.Diagnostics.AddError("error while tagging an entity", err.Error())
		}

		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("actions found: %d\n", len(actions)))

	for _, action := range actions {
		var tfAction ActionModel
		if err := CopyStructFields(ctx, action, &tfAction); err != nil {
			resp.Diagnostics.AddError("error coverting actions", err.Error())
		}

		state.Actions = append(state.Actions, tfAction)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

}

// Copy Struct copies the elements from one struct to another
func CopyStructFields(ctx context.Context, src interface{}, dst interface{}) error {
	srcVal := reflect.ValueOf(src)
	srcTyp := reflect.TypeOf(src)

	dstVal := reflect.ValueOf(dst)
	dstTyp := reflect.TypeOf(dst)

	if srcTyp.Kind() == reflect.Pointer {
		srcVal = srcVal.Elem()
	}

	if dstTyp.Kind() == reflect.Pointer {
		dstTyp = dstTyp.Elem()
		dstVal = dstVal.Elem()
	}

	if srcVal.Kind() != reflect.Struct || dstVal.Kind() != reflect.Struct {
		return fmt.Errorf("both source and destination must be structs")
	}

	for i := 0; i < dstTyp.NumField(); i++ {
		dstField := dstTyp.Field(i)
		dstValue := dstVal.Field(i)

		msg := fmt.Sprintf("setting field: %s of Type: %s with PkgPath of: %s\n", dstField.Name, dstField.Type, dstField.PkgPath)
		tflog.Trace(ctx, msg)

		srcFieldValue := srcVal.FieldByName(dstField.Name)
		if !srcFieldValue.IsValid() {
			msg := fmt.Sprintf("field Name: %s, is invalid: field value: %s\n", dstField.Name, srcFieldValue)
			tflog.Debug(ctx, msg)
			continue
		}
		if !dstValue.CanSet() {
			msg := fmt.Sprintf("field Name: %s, cannot be set: field value: %s\n", dstField.Name, srcFieldValue)
			tflog.Error(ctx, msg)
			continue
		}
		if err := copyField(ctx, srcFieldValue, dstValue); err != nil {
			msg := fmt.Sprintf("error converting field: %s of type: %s: %v", dstField.Name, dstField.Type, err)
			tflog.Error(ctx, msg)
		}

	}
	return nil
}

// copyField copies a single field based on the source type
func copyField(ctx context.Context, srcValue, dstValue reflect.Value) error {

	msg := fmt.Sprintf("Field Kind: %s", srcValue.Kind())
	tflog.Trace(ctx, msg)
	switch srcValue.Kind() {
	case reflect.String:
		dstValue.Set(reflect.ValueOf(types.StringValue(srcValue.String())))
	case reflect.Int64, reflect.Int:
		dstValue.Set(reflect.ValueOf(types.Int64Value(srcValue.Int())))
	case reflect.Float64:
		dstValue.Set(reflect.ValueOf(types.Float64Value(srcValue.Float())))
	case reflect.Float32:
		dstValue.Set(reflect.ValueOf(types.Float32Value(float32(srcValue.Float()))))
	case reflect.Bool:
		dstValue.Set(reflect.ValueOf(types.BoolValue(srcValue.Bool())))
	case reflect.Struct:
		dstPtr := dstValue.Addr()
		if err := CopyStructFields(ctx, srcValue.Interface(), dstPtr.Interface()); err != nil {
			return err
		}
	case reflect.Slice:
		srcType := srcValue.Type()

		// json.RawMessage is a slice under the hook so need to handle it separately
		if srcType == reflect.TypeOf(json.RawMessage{}) {
			dstValue.Set(reflect.ValueOf(getRawJson(srcValue)))
			return nil
		}
		msg := fmt.Sprintf("Slice Field Kind: %s", srcType.Elem().Kind())
		tflog.Trace(ctx, msg)

		switch srcType.Elem().Kind() {
		case reflect.Struct:
			newSlice, err := getNewSliceStruct(ctx, srcValue, dstValue.Type())
			if err != nil {
				return err
			}
			dstValue.Set(newSlice)

		case reflect.String:
			newSlice, err := getNewSliceString(srcValue, dstValue.Type())
			if err != nil {
				return err
			}
			dstValue.Set(newSlice)
		}

	case reflect.Map:
		mapValue := srcValue.Type().Elem()

		switch mapValue.Kind() {
		case reflect.String, reflect.Slice:
			dstValue.Set(reflect.ValueOf(srcValue.Interface()))
		default:
			msg := fmt.Sprintf("unsupported map field type: %s", srcValue.Kind())
			tflog.Warn(ctx, msg)
			return nil
		}

	default:
		msg := fmt.Sprintf("unsupported field type: %s", srcValue.Kind())
		tflog.Warn(ctx, msg)
		return nil
	}
	return nil
}

// Returns jsontypes.Normalized from a reflect.Value
func getRawJson(val reflect.Value) jsontypes.Normalized {
	raw := val.Interface().(json.RawMessage)
	if raw != nil {
		return jsontypes.NewNormalizedValue(string(raw))
	}
	return jsontypes.NewNormalizedNull()
}

// Returns a reflect.Value containing a copy of the provided slice of structs,
// with each element formatted according to Terraform's type conventions.
func getNewSliceStruct(ctx context.Context, src reflect.Value, dst reflect.Type) (reflect.Value, error) {
	newSlice := reflect.MakeSlice(dst, src.Len(), src.Cap())

	for i := 0; i < src.Len(); i++ {
		elmSrcValue := src.Index(i)
		elmDesttype := dst.Elem()
		dstPtr := reflect.New(elmDesttype)
		if err := CopyStructFields(ctx, elmSrcValue.Interface(), dstPtr.Interface()); err != nil {
			return reflect.Value{}, err
		}
		dstValue := dstPtr.Elem()
		newSlice.Index(i).Set(dstValue)
	}
	return newSlice, nil
}

// Returns a reflect.Value containing a copy of the provided slice of strings,
// with each string formatted using Terraform's types.String convention.
func getNewSliceString(src reflect.Value, dst reflect.Type) (reflect.Value, error) {
	newSlice := reflect.MakeSlice(dst, src.Len(), src.Cap())

	for i := 0; i < src.Len(); i++ {
		elmSrcValue := src.Index(i)
		newSlice.Index(i).Set(reflect.ValueOf(types.StringValue(elmSrcValue.String())))
	}
	return newSlice, nil
}
