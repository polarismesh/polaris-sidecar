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

import (
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	v3 "github.com/envoyproxy/go-control-plane/envoy/service/ratelimit/v3"
	"github.com/polarismesh/polaris-go"
	"google.golang.org/protobuf/types/known/structpb"
)

func New() (*RateLimitServer, error) {
	return nil, nil
}

type RateLimitServer struct {
	limiter polaris.LimitAPI
}

func (svr *RateLimitServer) ShouldRateLimit(req *v3.RateLimitRequest) *v3.RateLimitResponse {
	return &v3.RateLimitResponse{
		OverallCode:          0,
		Statuses:             []*v3.RateLimitResponse_DescriptorStatus{},
		ResponseHeadersToAdd: []*corev3.HeaderValue{},
		RequestHeadersToAdd:  []*corev3.HeaderValue{},
		RawBody:              []byte{},
		DynamicMetadata:      &structpb.Struct{Fields: map[string]*structpb.Value{"": {Kind: nil}}},
		Quota:                &v3.RateLimitResponse_Quota{Requests: 0, ExpirationSpecifier: nil, Id: ""},
	}
}

func (svr *RateLimitServer) Start() {
	
}
