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

package rls

type Config struct {
	Enable   bool     `yaml:"enable"`
	Network  string   `yaml:"-"`
	Address  string   `yaml"-"`
	BindPort uint32   `yaml:"port"`
	TLSInfo  *TLSInfo `yaml:"tls_info"`
}

const DefaultRLSAddress = "/var/run/polaris/ratelimit/rls.sock"

// TLSInfo tls 配置信息
type TLSInfo struct {
	// CertFile 服务端证书文件
	CertFile string `yaml:"cert_file"`
	// KeyFile CertFile 的密钥 key 文件
	KeyFile string `yaml"json:"key_file"`
}

// IsEmpty 检查 tls 配置信息是否为空 当证书和密钥同时存在时才不为空
func (t *TLSInfo) IsEmpty() bool {
	if t == nil {
		return true
	}
	if t.CertFile != "" && t.KeyFile != "" {
		return false
	}
	return true
}
