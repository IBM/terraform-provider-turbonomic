// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var tagType = map[string]attr.Type{
	OptimizedByTagName: types.StringType,
}

var _ function.Function = &GetTagFunction{}

type GetTagFunction struct{}

func NewGetTagFunction() function.Function {
	return &GetTagFunction{}
}

func (f *GetTagFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "get_tag"
}

func (f *GetTagFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Get turbonomic tag",
		Description: "Returns turbonomic tag - {turbonomic_optimized_by = \"turbonomic-terraform-provider\"} to mark the resource as optimized by Turbonomic provider",
		Parameters:  []function.Parameter{},
		Return: function.ObjectReturn{
			AttributeTypes: tagType,
		},
	}
}

// Returns the optimized by turbonomic provider tag
func (f *GetTagFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {

	tagValueObj, diags := types.ObjectValue(
		tagType,
		map[string]attr.Value{
			OptimizedByTagName: types.StringValue(OptimizedByTagValue),
		},
	)

	resp.Error = function.FuncErrorFromDiags(ctx, diags)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &tagValueObj))

}
