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

package meshproxy

import (
	"testing"

	"github.com/polarismesh/polaris-go"
	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/model"
)

type MeshTestSuit struct {
	r *envoyRegistry
}

func newMeshTestSuit(t *testing.T, conf *resolverConfig) (*MeshTestSuit, *MockConsumerAPI) {
	mockConsumer := &MockConsumerAPI{}
	r := &envoyRegistry{
		conf:     conf,
		consumer: mockConsumer,
		business: "",
	}

	return &MeshTestSuit{
		r: r,
	}, mockConsumer
}

type MockConsumerAPI struct {
	mockRetSupplier func() *model.ServicesResponse
}

// GetOneInstance 同步获取单个服务
func (m *MockConsumerAPI) GetOneInstance(req *polaris.GetOneInstanceRequest) (*model.OneInstanceResponse, error) {
	return nil, nil
}

// GetInstances 同步获取可用的服务列表
func (m *MockConsumerAPI) GetInstances(req *polaris.GetInstancesRequest) (*model.InstancesResponse, error) {
	return nil, nil
}

// GetAllInstances 同步获取完整的服务列表
func (m *MockConsumerAPI) GetAllInstances(req *polaris.GetAllInstancesRequest) (*model.InstancesResponse, error) {
	return nil, nil
}

// GetRouteRule 同步获取服务路由规则
func (m *MockConsumerAPI) GetRouteRule(req *polaris.GetServiceRuleRequest) (*model.ServiceRuleResponse, error) {
	return nil, nil
}

// UpdateServiceCallResult 上报服务调用结果
func (m *MockConsumerAPI) UpdateServiceCallResult(req *polaris.ServiceCallResult) error {
	return nil
}

// WatchService 订阅服务消息
func (m *MockConsumerAPI) WatchService(req *polaris.WatchServiceRequest) (*model.WatchServiceResponse, error) {
	return nil, nil
}

// GetServices 根据业务同步获取批量服务
func (m *MockConsumerAPI) GetServices(req *polaris.GetServicesRequest) (*model.ServicesResponse, error) {
	return m.mockRetSupplier(), nil
}

// InitCalleeService 初始化服务运行中需要的被调服务
func (m *MockConsumerAPI) InitCalleeService(req *polaris.InitCalleeServiceRequest) error {
	return nil
}

// WatchAllInstances 监听服务实例变更事件
func (m *MockConsumerAPI) WatchAllInstances(req *polaris.WatchAllInstancesRequest) (*model.WatchAllInstancesResponse, error) {
	return nil, nil
}

// WatchAllServices 监听服务列表变更事件
func (m *MockConsumerAPI) WatchAllServices(req *polaris.WatchAllServicesRequest) (*model.WatchAllServicesResponse, error) {
	return nil, nil
}

// Destroy 销毁API，销毁后无法再进行调用
func (m *MockConsumerAPI) Destroy() {
}

// SDKContext 获取SDK上下文
func (m *MockConsumerAPI) SDKContext() api.SDKContext {
	return nil
}
