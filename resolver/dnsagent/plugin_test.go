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
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/model"
	"github.com/polarismesh/polaris-go/pkg/model/local"
	"github.com/polarismesh/polaris-go/pkg/model/pb"
	v1 "github.com/polarismesh/polaris-go/pkg/model/pb/v1"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type TestConsumerAPI struct{}

func (ca *TestConsumerAPI) SDKContext() api.SDKContext {
	return nil
}

// 同步获取单个服务
func (ca *TestConsumerAPI) GetOneInstance(req *api.GetOneInstanceRequest) (*model.OneInstanceResponse, error) {
	return nil, nil
}

// 同步获取可用的服务列表
func (ca *TestConsumerAPI) GetInstances(req *api.GetInstancesRequest) (*model.InstancesResponse, error) {
	return &model.InstancesResponse{
		Instances: testInstances,
	}, nil
}

// 同步获取完整的服务列表
func (ca *TestConsumerAPI) GetAllInstances(req *api.GetAllInstancesRequest) (*model.InstancesResponse, error) {
	return nil, nil
}

// 同步获取服务路由规则
func (ca *TestConsumerAPI) GetRouteRule(req *api.GetServiceRuleRequest) (*model.ServiceRuleResponse, error) {
	return nil, nil
}

// 上报服务调用结果
func (ca *TestConsumerAPI) UpdateServiceCallResult(req *api.ServiceCallResult) error {
	return nil
}

//销毁API，销毁后无法再进行调用
func (ca *TestConsumerAPI) Destroy() {

}

//订阅服务消息
func (ca *TestConsumerAPI) WatchService(req *api.WatchServiceRequest) (*model.WatchServiceResponse, error) {
	return nil, nil
}

// 同步获取网格规则
func (ca *TestConsumerAPI) GetMeshConfig(req *api.GetMeshConfigRequest) (*model.MeshConfigResponse, error) {
	return nil, nil
}

// 同步获取网格
func (ca *TestConsumerAPI) GetMesh(req *api.GetMeshRequest) (*model.MeshResponse, error) {
	return nil, nil
}

// 根据业务同步获取批量服务
func (ca *TestConsumerAPI) GetServicesByBusiness(req *api.GetServicesRequest) (*model.ServicesResponse, error) {
	return nil, nil
}

//初始化服务运行中需要的被调服务
func (ca *TestConsumerAPI) InitCalleeService(req *api.InitCalleeServiceRequest) error {
	return nil
}

func Test_resolverDiscovery_ServeDNS(t *testing.T) {

	resolver := &resolverDiscovery{
		consumer: &TestConsumerAPI{},
		suffix:   "",
		dnsTtl:   10,
		config: &resolverConfig{
			RouteLabels: map[string]string{
				"": "",
			},
		},
	}

	resp := resolver.ServeDNS(context.Background(), dns.Question{
		Name:   "_xmpp._tcp.example.com.",
		Qtype:  dns.TypeSRV,
		Qclass: 0,
	})

	s, _ := json.Marshal(resp)
	t.Logf("%s", string(s))

	time.Sleep(time.Duration(time.Second))
}

var (
	testInstances = []model.Instance{
		pb.NewInstanceInProto(&v1.Instance{
			Host: &wrapperspb.StringValue{
				Value: "127.0.0.1",
			},
			Port: &wrapperspb.UInt32Value{
				Value: 10,
			},
			Priority: &wrapperspb.UInt32Value{
				Value: 10,
			},
			Weight: &wrapperspb.UInt32Value{
				Value: 10,
			},
		}, &model.ServiceKey{}, local.NewInstanceLocalValue()),
		pb.NewInstanceInProto(&v1.Instance{
			Host: &wrapperspb.StringValue{
				Value: "127.0.0.2",
			},
			Port: &wrapperspb.UInt32Value{
				Value: 20,
			},
			Priority: &wrapperspb.UInt32Value{
				Value: 20,
			},
			Weight: &wrapperspb.UInt32Value{
				Value: 20,
			},
		}, &model.ServiceKey{}, local.NewInstanceLocalValue()),
		pb.NewInstanceInProto(&v1.Instance{
			Host: &wrapperspb.StringValue{
				Value: "127.0.0.3",
			},
			Port: &wrapperspb.UInt32Value{
				Value: 30,
			},
			Priority: &wrapperspb.UInt32Value{
				Value: 30,
			},
			Weight: &wrapperspb.UInt32Value{
				Value: 30,
			},
		}, &model.ServiceKey{}, local.NewInstanceLocalValue()),
		pb.NewInstanceInProto(&v1.Instance{
			Host: &wrapperspb.StringValue{
				Value: "127.0.0.4",
			},
			Port: &wrapperspb.UInt32Value{
				Value: 40,
			},
			Priority: &wrapperspb.UInt32Value{
				Value: 40,
			},
			Weight: &wrapperspb.UInt32Value{
				Value: 40,
			},
		}, &model.ServiceKey{}, local.NewInstanceLocalValue()),
	}
)
