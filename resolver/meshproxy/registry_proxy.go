/**
 * Tencent is pleased to support the open source community by making CL5 available.
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

package meshproxy

import (
	"github.com/polarismesh/polaris-go"

	"github.com/polarismesh/polaris-sidecar/pkg/log"
)

type registry interface {
	GetCurrentNsService() (map[string]struct{}, error)
}

func newRegistry(conf *resolverConfig, consumer polaris.ConsumerAPI, business string) (registry, error) {
	r := &envoyRegistry{
		conf:     conf,
		consumer: consumer,
		business: business,
	}
	return r, nil
}

type envoyRegistry struct {
	conf     *resolverConfig
	consumer polaris.ConsumerAPI
	business string
}

func (r *envoyRegistry) GetCurrentNsService() (map[string]struct{}, error) {
	var services map[string]struct{}
	req := &polaris.GetServicesRequest{}
	req.Business = r.business
	resp, err := r.consumer.GetServices(&polaris.GetServicesRequest{})
	if nil != err {
		log.Errorf("[Mesh] fail to request services from polaris, %v", err)
		return nil, err
	}
	if len(resp.Value) == 0 {
		log.Infof("[Mesh] services is empty")
		return services, nil
	}
	services = make(map[string]struct{}, len(resp.GetValue()))
	for _, svc := range resp.GetValue() {
		// 这里必须全匹配的模式存储
		services[svc.Service+"."+svc.Namespace] = struct{}{}
	}
	return services, nil
}
