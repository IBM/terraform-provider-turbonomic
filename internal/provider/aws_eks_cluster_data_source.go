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

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	turboclient "github.com/IBM/turbonomic-go-client"
)

// eksClusterCloudServiceName is the cloudServiceName returned by the Turbonomic
// parking API for an EKS cluster entity.
const eksClusterCloudServiceName = "AMAZON_ELASTIC_KUBERNETES_SERVICE"

var (
	_ datasource.DataSource              = &awsEKSClusterDataSource{}
	_ datasource.DataSourceWithConfigure = &awsEKSClusterDataSource{}
)

// AwsEKSClusterEntityModel is the full data source state.
type AwsEKSClusterEntityModel struct {
	// Input fields
	Name         types.String `tfsdk:"name"`
	DefaultState types.String `tfsdk:"default_state"`

	// Output fields
	EntityUUID       types.String `tfsdk:"entity_uuid"`
	CurrentState     types.String `tfsdk:"current_state"`
	NewState         types.String `tfsdk:"new_state"`
	CloudProvider    types.String `tfsdk:"cloud_provider"`
	CloudServiceName types.String `tfsdk:"cloud_service_name"`
}

type awsEKSClusterDataSource struct {
	client *turboclient.Client
}

func NewAwsEKSClusterDataSource() datasource.DataSource {
	return &awsEKSClusterDataSource{}
}

func (d *awsEKSClusterDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_aws_eks_cluster"
}

func (d *awsEKSClusterDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves Turbonomic Smart Parking state and schedule recommendations for an AWS EKS cluster. " +
			"Returns the current parking state and, if a smart parking schedule is active, the recommended next state.",
		Attributes: map[string]schema.Attribute{
			// ── Input ──────────────────────────────────────────────────────────
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name of the EKS cluster as it appears in Turbonomic (e.g. `\"Kubernetes-myekscluster\"`).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"default_state": schema.StringAttribute{
				MarkdownDescription: "Fallback state to use when the cluster is not found in the Turbonomic parking API.",
				Optional:            true,
			},

			// ── Output ─────────────────────────────────────────────────────────
			"entity_uuid": schema.StringAttribute{
				MarkdownDescription: "Turbonomic UUID of the EKS cluster entity.",
				Computed:            true,
			},
			"current_state": schema.StringAttribute{
				MarkdownDescription: "Current parking state of the cluster (`RUNNING`, `STOPPED`, `STARTING`, `STOPPING`, etc.).",
				Computed:            true,
			},
			"new_state": schema.StringAttribute{
				MarkdownDescription: "Recommended parking state. If a smart parking schedule is active, this differs from `current_state`. Falls back to `current_state`, then `default_state`.",
				Computed:            true,
			},
			"cloud_provider": schema.StringAttribute{
				MarkdownDescription: "Cloud provider of the cluster (e.g. `\"AWS\"`).",
				Computed:            true,
			},
			"cloud_service_name": schema.StringAttribute{
				MarkdownDescription: "Cloud service name of the cluster (e.g. `\"AMAZON_ELASTIC_KUBERNETES_SERVICE\"`).",
				Computed:            true,
			},
		},
	}
}

func (d *awsEKSClusterDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*providerData)
	if !ok {
		resp.Diagnostics.AddError(
			"unexpected data-source configure type",
			fmt.Sprintf("expected: *providerData, got: %T. please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.client = data.Client.(*turboclient.Client)
}

func (d *awsEKSClusterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state AwsEKSClusterEntityModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		setDefaultsEKSClusterToNewState(&state)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	// ── 1. Search for the ContainerPlatformCluster entity ────────────────────
	uuid, errDetail := parkingSearchCluster(d.client, state.Name.ValueString())

	if errDetail == "" && uuid == "" {
		errDetail = fmt.Sprintf("EKS cluster %q not found in Turbonomic", state.Name.ValueString())
	}

	if errDetail != "" {
		tflog.Warn(ctx, errDetail)
		resp.Diagnostics.AddWarning("error while getting an entity", errDetail)
		setDefaultsEKSClusterToNewState(&state)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("aws eks cluster entity found: %s", uuid))

	// ── 2. Fetch parking state ────────────────────────────────────────────────
	parking, err := d.client.GetParkingEntity(uuid)
	if err != nil {
		errDetail = fmt.Sprintf("error fetching parking state for cluster %q: %s", state.Name.ValueString(), err.Error())
		tflog.Warn(ctx, errDetail)
		resp.Diagnostics.AddWarning("error while getting an entity", errDetail)
		setDefaultsEKSClusterToNewState(&state)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	// Validate this is an EKS cluster (safety check on cloudServiceName)
	if parking.CloudServiceName != eksClusterCloudServiceName {
		errDetail = fmt.Sprintf("cluster %q has cloudServiceName %q, expected %q", state.Name.ValueString(), parking.CloudServiceName, eksClusterCloudServiceName)
		tflog.Warn(ctx, errDetail)
		resp.Diagnostics.AddWarning("unexpected cluster cloud service", errDetail)
		setDefaultsEKSClusterToNewState(&state)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	// ── 3. Populate state ─────────────────────────────────────────────────────
	state.EntityUUID = types.StringValue(uuid)
	state.CurrentState = types.StringValue(parking.State)
	state.CloudProvider = types.StringValue(parking.Provider)
	state.CloudServiceName = types.StringValue(parking.CloudServiceName)
	state.NewState = types.StringValue(parkingNewState(parking.State, parking.SmartParkingRecommendation))

	// ── 4. Fallback chain ─────────────────────────────────────────────────────
	setDefaultsEKSClusterToNewState(&state)

	if err := TagEntity(d.client, uuid); err != nil {
		resp.Diagnostics.AddWarning("error while tagging an entity", err.Error())
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func setDefaultsEKSClusterToNewState(state *AwsEKSClusterEntityModel) {
	state.NewState = applyDefaultIfEmptyGeneric(state.NewState, state.CurrentState)
	state.NewState = applyDefaultIfEmptyGeneric(state.NewState, state.DefaultState)
	if state.EntityUUID.IsNull() {
		state.EntityUUID = types.StringValue("")
	}
	if state.CurrentState.IsNull() {
		state.CurrentState = types.StringValue("")
	}
	if state.CloudProvider.IsNull() {
		state.CloudProvider = types.StringValue("")
	}
	if state.CloudServiceName.IsNull() {
		state.CloudServiceName = types.StringValue("")
	}
}
