// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// filterMapping is one entry in the shorthand lookup table.
type filterMapping struct {
	Entity     string
	Field      string
	FilterType string
}

// filterMappings is the single source of truth for all shorthand combinations.
// To add entries: append one filterMapping here - nothing else changes.
var filterMappings = []filterMapping{
	// VirtualMachine
	{Entity: "vm", Field: "name", FilterType: "vmsByName"},
	{Entity: "vm", Field: "tag", FilterType: "vmsByTag"},
	{Entity: "vm", Field: "state", FilterType: "vmsByState"},
	{Entity: "vm", Field: "guest_os", FilterType: "vmsByGuestName"},
	{Entity: "vm", Field: "cloud_provider", FilterType: "vmsByCloudProvider"},
	{Entity: "vm", Field: "pm", FilterType: "vmsByPMName"},
	{Entity: "vm", Field: "cluster", FilterType: "vmsByClusterName"},
	{Entity: "vm", Field: "dc", FilterType: "vmsByDCnested"},
	{Entity: "vm", Field: "vdc", FilterType: "vmsByVDC"},
	{Entity: "vm", Field: "network", FilterType: "vmsByNetwork"},
	{Entity: "vm", Field: "storage", FilterType: "vmsByStorage"},
	{Entity: "vm", Field: "business_account", FilterType: "vmsByBusinessAccountUuid"},
	// PhysicalMachine
	{Entity: "pm", Field: "name", FilterType: "pmsByName"},
	{Entity: "pm", Field: "tag", FilterType: "pmsByTag"},
	{Entity: "pm", Field: "state", FilterType: "pmsByState"},
	{Entity: "pm", Field: "cluster", FilterType: "pmsByClusterName"},
	// Storage
	{Entity: "storage", Field: "name", FilterType: "storageByName"},
	{Entity: "storage", Field: "tag", FilterType: "storageByTag"},
	{Entity: "storage", Field: "state", FilterType: "storageByState"},
	{Entity: "storage", Field: "pm_cluster", FilterType: "storageByPMCluster"},
	// Database
	{Entity: "db", Field: "name", FilterType: "databaseByName"},
	{Entity: "db", Field: "tag", FilterType: "databaseByTag"},
	{Entity: "db", Field: "cloud_provider", FilterType: "databaseByCloudProvider"},
	{Entity: "db", Field: "business_account", FilterType: "databaseByBusinessAccountUuid"},
	// DatabaseServer
	{Entity: "db_server", Field: "name", FilterType: "databaseServerByName"},
	{Entity: "db_server", Field: "tag", FilterType: "databaseServerByTag"},
	{Entity: "db_server", Field: "business_account", FilterType: "databaseServerByBusinessAccountUuid"},
	// Container
	{Entity: "container", Field: "name", FilterType: "containersByName"},
	{Entity: "container", Field: "namespace", FilterType: "containersByNamespace"},
	// Namespace
	{Entity: "namespace", Field: "name", FilterType: "namespacesByName"},
	// Cluster (compute)
	{Entity: "cluster", Field: "name", FilterType: "clustersByName"},
	{Entity: "cluster", Field: "tag", FilterType: "clustersByTag"},
	// StorageCluster
	{Entity: "storage_cluster", Field: "name", FilterType: "storageClustersByName"},
	// VirtualMachineCluster
	{Entity: "vm_cluster", Field: "name", FilterType: "virtualMachineClustersByName"},
	// ContainerPlatformCluster
	{Entity: "container_platform_cluster", Field: "name", FilterType: "containerPlatformClustersByName"},
	// ResourceGroup
	{Entity: "resource_group", Field: "name", FilterType: "resourceGroupByName"},
	{Entity: "resource_group", Field: "tag", FilterType: "resourceGroupByTag"},
	{Entity: "resource_group", Field: "business_account", FilterType: "resourceGroupByBusinessAccountUuid"},
	// BusinessAccount
	{Entity: "business_account", Field: "name", FilterType: "businessAccountByName"},
	// Volume
	{Entity: "volume", Field: "storage", FilterType: "volumeByStorage"},
	{Entity: "volume", Field: "resource_group", FilterType: "volumeByResourceGroupName"},
}

// Derived maps - populated by init() from filterMappings.
var filterShorthandMap map[string]map[string]string // [entity][field] → filterType
var filterTypeToShorthand map[string][2]string      // filterType → [entity, field]
var validFilterEntities []string                    // sorted unique entity aliases
var validFilterFields []string                      // sorted unique field names

func init() {
	filterShorthandMap = make(map[string]map[string]string)
	filterTypeToShorthand = make(map[string][2]string)

	seenEntities := make(map[string]struct{})
	seenFields := make(map[string]struct{})

	for _, m := range filterMappings {
		if filterShorthandMap[m.Entity] == nil {
			filterShorthandMap[m.Entity] = make(map[string]string)
		}
		filterShorthandMap[m.Entity][m.Field] = m.FilterType
		filterTypeToShorthand[m.FilterType] = [2]string{m.Entity, m.Field}

		seenEntities[m.Entity] = struct{}{}
		seenFields[m.Field] = struct{}{}
	}

	for e := range seenEntities {
		validFilterEntities = append(validFilterEntities, e)
	}
	for f := range seenFields {
		validFilterFields = append(validFilterFields, f)
	}
	sort.Strings(validFilterEntities)
	sort.Strings(validFilterFields)
}

// resolveCriteriaFilterType applies the priority rules to derive the API filterType string.
// Returns (filterTypeStr, true) on success, ("", false) if an error was added to diags.
func resolveCriteriaFilterType(block criteriaListModel, diags *diag.Diagnostics) (string, bool) {
	rawFilterType := block.FilterType.ValueString()
	rawFilterTypeSet := !block.FilterType.IsNull() && !block.FilterType.IsUnknown() && rawFilterType != ""

	entity := block.FilterEntity.ValueString()
	field := block.FilterField.ValueString()
	shorthandSet := (!block.FilterEntity.IsNull() && entity != "") ||
		(!block.FilterField.IsNull() && field != "")

	// Rule 1: filter_type set AND shorthand also set.
	if rawFilterTypeSet && shorthandSet {
		// If filter_type matches what the shorthand would resolve to, it was set by ModifyPlan
		// (or the user provided a consistent value). Either way, no conflict - use it silently.
		if !block.FilterEntity.IsNull() && entity != "" && !block.FilterField.IsNull() && field != "" {
			if fieldMap, ok := filterShorthandMap[entity]; ok {
				if ft, ok := fieldMap[field]; ok && ft == rawFilterType {
					return rawFilterType, true
				}
			}
		}
		// Genuine conflict: user set both filter_type and shorthand with different intent. Warn.
		diags.AddWarning(
			"filter_entity and filter_field ignored",
			"filter_entity and filter_field are ignored because filter_type is explicitly set. "+
				"Remove filter_entity and filter_field, or remove filter_type to use the shorthand.",
		)
		return rawFilterType, true
	}

	// Rule 2: filter_type only → use directly
	if rawFilterTypeSet {
		return rawFilterType, true
	}

	// Rule 3: both shorthand fields set → resolve from lookup table
	if !block.FilterEntity.IsNull() && entity != "" && !block.FilterField.IsNull() && field != "" {
		if fieldMap, ok := filterShorthandMap[entity]; ok {
			if ft, ok := fieldMap[field]; ok {
				return ft, true
			}
		}
		diags.AddError(
			"unsupported filter_entity + filter_field combination",
			fmt.Sprintf(
				"filter_entity %q + filter_field %q is not a supported combination. "+
					"Use filter_type directly for unlisted filters.",
				entity, field,
			),
		)
		return "", false
	}

	// Rule 4: neither set
	diags.AddError(
		"criteria_list requires filter_type or shorthand",
		"criteria_list requires either filter_type, or both filter_entity and filter_field.",
	)
	return "", false
}

// stringPtr returns a pointer to the given string value.
func stringPtr(s string) *string {
	return &s
}
