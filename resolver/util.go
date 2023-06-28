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
	"strings"

	"github.com/polarismesh/polaris-go/pkg/config"
	"github.com/polarismesh/polaris-go/pkg/model"
)

const (
	Quota        = "."
	sysNamespace = "polaris"
)

var (
	ContextProtocol = struct{}{}
)

// ParseQname parse the qname into service and suffix
// qname format: <service>.<namespace>.<suffix>
func ParseQname(qname string, suffix string, currentNs string) *model.ServiceKey {
	var matched bool
	qname, matched = MatchSuffix(qname, suffix)
	if !matched {
		return nil
	}
	if strings.HasSuffix(qname, Quota) {
		qname = qname[0 : len(qname)-1]
	}
	var namespace string
	var serviceName string
	sepIndex := strings.LastIndex(qname, Quota)
	if sepIndex < 0 {
		namespace = currentNs
		serviceName = qname
	} else {
		namespace = qname[sepIndex+1:]
		if strings.ToLower(namespace) == sysNamespace {
			namespace = config.ServerNamespace
		}
		serviceName = qname[:sepIndex]
	}
	return &model.ServiceKey{Namespace: namespace, Service: serviceName}
}

// MatchSuffix match the suffix and return the split qname
func MatchSuffix(qname string, suffix string) (string, bool) {
	if len(suffix) > 0 && !strings.HasSuffix(qname, suffix) {
		return qname, false
	}
	if len(suffix) > 0 {
		qname = qname[:len(qname)-len(suffix)]
		return qname, true
	}
	return qname, true
}
