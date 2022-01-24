/**
 * Tencent is pleased to support the open source community by making Polaris available.
 *
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 *
 * Licensed under the BSD 3-Clause License (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://opensource.org/licenses/BSD-3-Clause
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package resolver

import (
	"context"

	"github.com/miekg/dns"
)

const CtxKeyProtocol = "protocol"

// ConfigEntry: resolver plugin config entry
type ConfigEntry struct {
	Name   string                 `yaml:"name"`
	Suffix string                 `yaml:"suffix"`
	DnsTtl int                    `yaml:"dns_ttl"`
	Enable bool                   `yaml:"enable"`
	Option map[string]interface{} `yaml:"option"`
}

// NamingResolver resolver interface
type NamingResolver interface {
	// Name will return the name to resolver
	Name() string
	// Initialize will init the resolver on startup
	Initialize(c *ConfigEntry) error
	// Start the plugin runnable
	Start(context.Context)
	// Destroy will destroy the resolver on shutdown
	Destroy()
	// ServeDNS is like dns.Handler except ServeDNS may return an response or nil
	ServeDNS(context.Context, dns.Question) *dns.Msg
}

var resolvers = map[string]NamingResolver{}

// Register naming resolver
func Register(namingResolver NamingResolver) {
	resolvers[namingResolver.Name()] = namingResolver
}

// NameResolver get the resolver by name
func NameResolver(name string) NamingResolver {
	return resolvers[name]
}
