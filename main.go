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

package main

import (
	"context"
	"flag"
	"log"

	"github.com/IBM/terraform-provider-turbonomic/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

const (
	version           = "1.2.0"
	typeName          = "turbonomic"
	tfProviderAddress = "registry.terraform.io/IBM/turbonomic"
)

var debug bool

func main() {
	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: tfProviderAddress,
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version, typeName), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
