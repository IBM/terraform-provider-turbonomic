// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	turboclient "github.com/IBM/turbonomic-go-client"
	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type ebsVolumeType string

const (
	standard ebsVolumeType = "Standard (Legacy)"
	gp2      ebsVolumeType = "General Purpose SSD (2nd-gen)"
	gp3      ebsVolumeType = "General Purpose SSD (3rd-gen)"
	io1      ebsVolumeType = "Provisioned IOPS SSD (1st-gen)"
	io2      ebsVolumeType = "Provisioned IOPS SSD (2nd-gen)"
	sc1      ebsVolumeType = "Cold HDD"
	st1      ebsVolumeType = "Throughput Optimized HDD"
)

var (
	ebsTypeMap = map[ebsVolumeType]string{
		standard: "standard",
		gp2:      "gp2",
		gp3:      "gp3",
		io1:      "io1",
		io2:      "io2",
		sc1:      "sc1",
		st1:      "st1",
	}
	ebsVolumeTypes = slices.Collect(maps.Values(ebsTypeMap))
)

const (
	EBS_VOL_DATASOURCE_NAME = "aws_ebs_volume"
)

type ebsVolumeEntityModel struct {
	UUID              types.String `tfsdk:"entity_uuid"`
	Name              types.String `tfsdk:"entity_name"`
	Type              types.String `tfsdk:"entity_type"`
	CurrentVolumeType types.String `tfsdk:"current_type"`
	NewVolumeType     types.String `tfsdk:"new_type"`
	DefaultVolumeType types.String `tfsdk:"default_type"`
}

type ebsVolumeSource struct {
	client *turboclient.Client
}

func NewEBSVolumeDataSource() datasource.DataSource {
	return &ebsVolumeSource{}
}

func (d *ebsVolumeSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + EBS_VOL_DATASOURCE_NAME
}

func (d *ebsVolumeSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The following example demonstrates the syntax for the `turbonomic_aws_ebs_volume` volume source. It uses Turbonomic engine to recommend AWS EBS volume type. For more info, see the [ebs documentation](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/ebs_volume).",
		Attributes: map[string]schema.Attribute{
			"entity_uuid": schema.StringAttribute{
				MarkdownDescription: "turbonomic uuid of the volume entity",
				Computed:            true,
			},
			"entity_name": schema.StringAttribute{
				MarkdownDescription: "name of the volume entity",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"entity_type": schema.StringAttribute{
				MarkdownDescription: "type of the volume entity",
				Computed:            true,
			},
			"default_type": schema.StringAttribute{
				MarkdownDescription: "default type of the volume entity",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive(ebsVolumeTypes...),
				},
			},
			"current_type": schema.StringAttribute{
				MarkdownDescription: "current type of the volume entity",
				Computed:            true,
			},
			"new_type": schema.StringAttribute{
				MarkdownDescription: "recommended type of the volume entity",
				Computed:            true,
			},
		},
	}
}

func (d ebsVolumeSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.RequiredTogether(
			path.MatchRoot("default_type"),
		),
	}
}

func (d *ebsVolumeSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ebsVolumeSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ebsVolumeEntityModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	state.Type = types.StringValue("VirtualVolume")

	entityName, entityType := state.Name.ValueString(), state.Type.ValueString()
	entity, errDiag := GetEntitiesByNameAndType(d.client, entityName, entityType, "CLOUD", "AWS")
	if errDiag != nil {
		tflog.Error(ctx, errDiag.Detail())
		resp.Diagnostics.AddError(errDiag.Summary(), errDiag.Detail())
		return
	} else if len(entity) == 0 {
		errDetail := fmt.Sprintf("Entity %s of type %s not found in Turbonomic instance", entityName, entityType)
		tflog.Warn(ctx, errDetail)

		// if entity doesn't exist, update new volume type
		state = d.populateNewType(state)

		state.Type = types.StringNull()

		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Entity id found: %s\n", entity[0].UUID))
	state.UUID = types.StringValue(entity[0].UUID)

	actions, errDiag := GetActionsByEntityUUIDAndType(d.client, entity[0].UUID, "SCALE")
	if errDiag != nil {
		tflog.Error(ctx, errDiag.Detail())
		resp.Diagnostics.AddError(errDiag.Summary(), errDiag.Detail())
		return
	} else if len(actions) == 0 {
		errDetail := fmt.Sprintf("no matching action found for entity id: %s", entity[0].UUID)
		tflog.Trace(ctx, errDetail)

		// if action doesn't exist, update curr and new volume type
		state = d.populateCurrType(state)
		state = d.populateNewType(state)

		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	state.Type = types.StringValue("VirtualVolume")

	// TODO: Fix trace vs debug vs warn errors
	tflog.Debug(ctx, fmt.Sprintf("Action id found: %d\n", actions[0].ActionID))

	// for each compound action, update the state
	// for now it will only pickup storage tier update in compound action and update the state
	for _, act := range actions[0].CompoundActions {
		if len(act.CurrentEntity.DisplayName) == 0 || len(act.NewEntity.DisplayName) == 0 {
			errDetail := "error while parsing action dto"
			tflog.Error(ctx, errDetail)
			resp.Diagnostics.AddError("turbonomic api error", errDetail)
			continue
		}

		currVolType := act.CurrentEntity.DisplayName
		newVolType := act.NewEntity.DisplayName

		state.CurrentVolumeType = types.StringValue(strings.ToLower(currVolType))
		state.NewVolumeType = types.StringValue(strings.ToLower(newVolType))
	}

	// if compound actions doesn't have storage tier update, use default type
	state = d.populateCurrType(state)
	state = d.populateNewType(state)

	if err := TagEntity(d.client, state.UUID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error while tagging an entity", err.Error())
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// update new-volume-type with default-type if it's still empty
func (*ebsVolumeSource) populateNewType(state ebsVolumeEntityModel) ebsVolumeEntityModel {
	if (state.NewVolumeType.IsNull() || len(state.NewVolumeType.ValueString()) == 0) && !state.DefaultVolumeType.IsNull() {
		state.NewVolumeType = state.DefaultVolumeType
	}
	return state
}

// update curr-volume-type with default-type if it's still empty
func (*ebsVolumeSource) populateCurrType(state ebsVolumeEntityModel) ebsVolumeEntityModel {
	if (state.CurrentVolumeType.IsNull() || len(state.CurrentVolumeType.ValueString()) == 0) && !state.DefaultVolumeType.IsNull() {
		state.CurrentVolumeType = state.DefaultVolumeType
	}
	return state
}
