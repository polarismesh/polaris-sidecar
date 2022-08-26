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

package dnsagent

import (
	"encoding/json"
	"fmt"
	"strings"
)

type resolverConfig struct {
	RouteLabelsMap map[string]string `json:"-"`
	RouteLabels    string            `json:"route_labels"`
}

func parseLabels(value string) map[string]string {
	values := make(map[string]string)
	if len(value) == 0 {
		return values
	}
	tokens := strings.Split(value, ",")
	for _, token := range tokens {
		idx := strings.Index(token, ":")
		if idx < 0 {
			values[token] = token
		} else {
			values[token[0:idx]] = token[idx+1:]
		}
	}
	return values
}

func parseOptions(options map[string]interface{}) (*resolverConfig, error) {
	config := &resolverConfig{}
	if len(options) == 0 {
		return config, nil
	}
	jsonBytes, err := json.Marshal(options)
	if nil != err {
		return nil, fmt.Errorf("fail to marshal %s config entry, err is %v", name, err)
	}
	if err = json.Unmarshal(jsonBytes, config); nil != err {
		return nil, fmt.Errorf("fail to unmarshal %s config entry, err is %v", name, err)
	}
	config.RouteLabelsMap = parseLabels(config.RouteLabels)
	return config, nil
}
