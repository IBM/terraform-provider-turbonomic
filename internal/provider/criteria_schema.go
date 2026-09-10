// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// criteriaAttrTypes is the canonical attr.Type map for a single criteria object.
// Used when building object values, list types, and schema definitions.
// All consumers (turbonomic_filter, turbonomic_group) reference this single map.
func criteriaAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"filter_entity":  types.StringType,
		"filter_field":   types.StringType,
		"filter_type":    types.StringType,
		"operator":       types.StringType,
		"value":          types.StringType,
		"case_sensitive": types.BoolType,
	}
}

// criteriaObjectType returns the ObjectType for a single criteria entry.
func criteriaObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: criteriaAttrTypes()}
}

// criteriaNestedBlockObject returns the shared schema.NestedBlockObject used by both
// turbonomic_group (as a ListNestedBlock) and turbonomic_filter (as a ListNestedBlock).
func criteriaNestedBlockObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"filter_entity": schema.StringAttribute{
				Description: "Short alias for the entity type to filter on. " +
					"Valid values: vm, pm, storage, db, db_server, container, namespace, cluster, " +
					"storage_cluster, vm_cluster, container_platform_cluster, resource_group, " +
					"business_account, volume. " +
					"Use together with filter_field. Ignored if filter_type is also set.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.OneOf(validFilterEntities...),
				},
			},
			"filter_field": schema.StringAttribute{
				Description: "The property to filter by. " +
					"Valid values: name, tag, state, guest_os, cloud_provider, pm, cluster, dc, vdc, " +
					"network, storage, business_account, pm_cluster, namespace, resource_group. " +
					"Use together with filter_entity.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.OneOf(validFilterFields...),
				},
			},
			"filter_type": schema.StringAttribute{
				Description: "Internal Turbonomic filter key resolved from filter_entity and filter_field. " +
					"Set this only if the desired combination is not covered by filter_entity/filter_field.",
				Optional: true,
				Computed: true,
			},
			"operator": schema.StringAttribute{
				Description: "Comparison operator. " +
					"String filters: equals, not_equals, regex, not_regex. " +
					"Numeric filters: greater_than, less_than, greater_than_equals, less_than_equals, equals, not_equals. " +
					"Selection filters: equals, not_equals, exists, not_exists.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"equals", "not_equals",
						"greater_than", "less_than", "greater_than_equals", "less_than_equals",
						"regex", "not_regex",
						"exists", "not_exists",
					),
				},
			},
			"value": schema.StringAttribute{
				Description: "Value to match against. Not required when operator is exists or not_exists.",
				Optional:    true,
			},
			"case_sensitive": schema.BoolAttribute{
				Description: "Whether string/regex matching is case-sensitive. " +
					"Defaults to false (case-insensitive). Only affects string and regex operators.",
				Optional: true,
			},
		},
	}
}

// criteriaListBlock returns the complete schema.ListNestedBlock used in both
// turbonomic_group and turbonomic_filter.
func criteriaListBlock() schema.Block {
	return schema.ListNestedBlock{
		Description:  "Criteria for dynamic group membership or filter definition. Each block is one filter criterion.",
		NestedObject: criteriaNestedBlockObject(),
	}
}

// buildCriteriaObject constructs a types.Object for a single criteriaListModel value.
// The resulting object conforms to criteriaAttrTypes().
func buildCriteriaObject(m criteriaListModel) (types.Object, error) {
	obj, diags := types.ObjectValue(criteriaAttrTypes(), map[string]attr.Value{
		"filter_entity":  m.FilterEntity,
		"filter_field":   m.FilterField,
		"filter_type":    m.FilterType,
		"operator":       m.Operator,
		"value":          m.Value,
		"case_sensitive": m.CaseSensitive,
	})
	if diags.HasError() {
		return types.ObjectNull(criteriaAttrTypes()), fmt.Errorf("building criteria object: %s", diags[0].Detail())
	}
	return obj, nil
}
