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
	"context"
	"net"
	"strings"

	v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/common/ratelimit/v3"
	pb "github.com/envoyproxy/go-control-plane/envoy/service/ratelimit/v3"
	"github.com/polarismesh/polaris-go"
	"github.com/polarismesh/polaris-go/pkg/model"
	"github.com/polarismesh/polaris-sidecar/pkg/client"
	"github.com/polarismesh/polaris-sidecar/pkg/log"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func New(namespace string, conf *Config) (*RateLimitServer, error) {
	return &RateLimitServer{
		namespace: namespace,
		conf:      conf,
	}, nil
}

type RateLimitServer struct {
	namespace string
	conf      *Config
	ln        net.Listener
	grpcSvr   *grpc.Server
	limiter   polaris.LimitAPI
}

func (svr *RateLimitServer) Run(ctx context.Context) error {
	ln, err := net.Listen(svr.conf.Network, svr.conf.Address)
	if err != nil {
		return err
	}
	svr.ln = ln
	svr.limiter, err = client.GetLimitAPI()
	if err != nil {
		return err
	}

	// 指定使用服务端证书创建一个 TLS credentials
	var creds credentials.TransportCredentials
	if !svr.conf.TLSInfo.IsEmpty() {
		creds, err = credentials.NewServerTLSFromFile(svr.conf.TLSInfo.CertFile, svr.conf.TLSInfo.KeyFile)
		if err != nil {
			return err
		}
	}
	// 设置 grpc server options
	opts := []grpc.ServerOption{}
	if creds != nil {
		// 指定使用 TLS credentials
		opts = append(opts, grpc.Creds(creds))
	}
	server := grpc.NewServer(opts...)
	pb.RegisterRateLimitServiceServer(server, svr)
	return server.Serve(ln)
}

func (svr *RateLimitServer) Destroy() {
	if svr.grpcSvr != nil {
		svr.grpcSvr.GracefulStop()
	}
	if svr.ln != nil {
		_ = svr.ln.Close()
	}
}

const MaxUint32 = uint32(1<<32 - 1)

func (svr *RateLimitServer) ShouldRateLimit(ctx context.Context, req *pb.RateLimitRequest) (*pb.RateLimitResponse, error) {
	log.Info("[envoy-rls] receive ratelimit request", zap.Any("req", req))
	acquireQuota := req.GetHitsAddend()
	if acquireQuota == 0 {
		acquireQuota = 1
	}

	quotaReq, err := svr.buildQuotaRequest(req.GetDomain(), acquireQuota, req.GetDescriptors())
	if err != nil {
		log.Error("[envoy-rls] build ratelimit quota request", zap.Error(err))
		return nil, err
	}
	future, err := svr.limiter.GetQuota(quotaReq)
	if err != nil {
		log.Error("[envoy-rls] get quota", zap.Error(err))
		return nil, err
	}
	resp := future.Get()

	overallCode := pb.RateLimitResponse_OK
	if resp.Code == model.QuotaResultLimited {
		overallCode = pb.RateLimitResponse_OVER_LIMIT
	}

	descriptorStatus := make([]*pb.RateLimitResponse_DescriptorStatus, 0, len(req.GetDescriptors()))
	for range req.GetDescriptors() {
		descriptorStatus = append(descriptorStatus, &pb.RateLimitResponse_DescriptorStatus{
			Code: overallCode,
		})
	}

	rlsRsp := &pb.RateLimitResponse{
		OverallCode: overallCode,
		Statuses:    descriptorStatus,
		RawBody:     []byte(resp.Info),
	}
	log.Info("[envoy-rls] send envoy rls response", zap.Any("rsp", rlsRsp))
	return rlsRsp, nil
}

func (svr *RateLimitServer) buildQuotaRequest(domain string, acquireQuota uint32,
	descriptors []*v3.RateLimitDescriptor) (polaris.QuotaRequest, error) {

	req := polaris.NewQuotaRequest()
	for i := range descriptors {
		descriptor := descriptors[i]
		for _, entry := range descriptor.GetEntries() {
			if entry.GetKey() == ":path" {
				req.SetMethod(entry.GetValue())
				continue
			}
			req.AddArgument(model.BuildArgumentFromLabel(entry.GetKey(), entry.GetValue()))
		}
	}

	if strings.HasSuffix(domain, "."+svr.namespace) {
		domain = strings.TrimSuffix(domain, "."+svr.namespace)
	}
	req.SetNamespace(svr.namespace)
	req.SetService(domain)
	req.SetToken(acquireQuota)

	return req, nil
}
