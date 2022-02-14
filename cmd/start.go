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

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/polarismesh/polaris-sidecar/bootstrap"
)

var (
	configFilePath = ""

	bootConfig bootstrap.BootConfig

	startCmd = &cobra.Command{
		Use:   "start",
		Short: "start running",
		Long:  "start running",
		Run: func(c *cobra.Command, args []string) {
			bootstrap.Start(configFilePath, &bootConfig)
		},
	}
)

/**
 * @brief 解析命令参数
 */
func init() {
	startCmd.PersistentFlags().StringVarP(
		&configFilePath, "config-file", "c", "polaris-sidecar.yaml", "config file path")
	startCmd.PersistentFlags().StringVarP(
		&bootConfig.Bind, "bind", "b", "", "polaris sidecar bind host")
	startCmd.PersistentFlags().IntVarP(
		&bootConfig.Port, "port", "p", 0, "polaris sidecar listen port")
	startCmd.PersistentFlags().StringVarP(
		&bootConfig.LogLevel, "log-level", "l", "", "polaris sidecar logger level")
	startCmd.PersistentFlags().StringVarP(&bootConfig.RecurseEnabled,
		"recurse-enabled", "re", "", "polaris sidecar recurse enabled")
	startCmd.PersistentFlags().StringVarP(&bootConfig.ResolverDnsAgentEnabled,
		"dnsagent-enabled", "de", "", "polaris sidecar resolver dnsagent enabled")
	startCmd.PersistentFlags().StringVarP(&bootConfig.ResolverDnsAgentRouteLabels,
		"dnsagent-route-labels", "drl", "", "polaris sidecar resolver dnsagent route lables")
	startCmd.PersistentFlags().StringVarP(&bootConfig.ResolverMeshProxyEnabled,
		"meshproxy-enabled", "me", "", "polaris sidecar resolver mesh proxy enabled")
}
