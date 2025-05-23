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
	"github.com/IBM/terraform-provider-turbonomic/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

const (
	// user specific fields to be set when checking acceptance testing
	// use local/fyre turbo app details and a VM name available in the app

	providerConfig = `
        provider "turbonomic" {
            username = "<username>"
            password = "<password>"
            hostname = "<hostname>"
            skipverify = true
        }
	`
	providerConfigoAuth = `
        provider "turbonomic" {
            client_id = "<client_id>"
            client_secret = "<client_secret>"
            role = "<role>"
            hostname = "<hostname>"
            skipverify = true
        }
	`
	vmName     = "<vm-name>"
	vmCurrSize = "<vmCurrSize>"
	vmNewSize  = "<vmNewSize>"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"turbonomic": providerserver.NewProtocol6WithError(provider.New("test", "turbonomic")()),
}
