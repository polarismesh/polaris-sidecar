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

package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	adminv3 "github.com/envoyproxy/go-control-plane/envoy/admin/v3"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/polarismesh/polaris-go"
	"github.com/polarismesh/polaris-go/pkg/config"
	"github.com/polarismesh/polaris-go/pkg/model"
	"github.com/polarismesh/polaris-go/pkg/model/pb"
	namingpb "github.com/polarismesh/polaris-go/pkg/model/pb/v1"
	"github.com/polarismesh/polaris-go/plugin/statreporter/prometheus"
	"github.com/polarismesh/polaris-sidecar/log"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type Server struct {
	port      int
	consumer  polaris.ConsumerAPI
	namespace string
}

func NewServer(namespace string, port int) *Server {
	srv := &Server{
		namespace: namespace,
		port:      port,
	}
	return srv
}

type InstanceMetricKey struct {
	ClusterName string
	Host        string
	Port        uint32
}

func (i InstanceMetricKey) String() string {
	return fmt.Sprintf("ClusterName %s, Host %s, Port %d", i.ClusterName, i.Host, i.Port)
}

type InstanceMetricValue struct {
	RqSuccess uint64
	RqError   uint64
	RqTotal   uint64
}

func (i InstanceMetricValue) String() string {
	return fmt.Sprintf("RqSuccess %d, RqError %d, RqTotal %d", i.RqSuccess, i.RqError, i.RqTotal)
}

const ticketDuration = 30 * time.Second

func (s *Server) Start(ctx context.Context) error {
	var err error
	cfg := config.NewDefaultConfigurationWithDomain()
	cfg.GetConsumer().GetCircuitBreaker().SetEnable(false)
	cfg.GetGlobal().GetStatReporter().SetEnable(true)
	cfg.GetGlobal().GetStatReporter().SetChain([]string{prometheus.PluginName})
	_ = cfg.GetGlobal().GetStatReporter().SetPluginConfig(prometheus.PluginName, &prometheus.Config{
		Port: s.port,
	})
	s.consumer, err = polaris.NewConsumerAPIByConfig(cfg)
	if nil != err {
		return err
	}
	ticker := time.NewTicker(ticketDuration)
	values := make(map[InstanceMetricKey]*InstanceMetricValue)
	for {
		select {
		case <-ticker.C:
			s.reportMetricByCluster(values)
		case <-ctx.Done():
			log.Errorf("Server metric service stopped")
			return nil
		}
	}
}

func (s *Server) getClusterStats() *StatsObject {
	resp, err := http.Get("http://127.0.0.1:15000/stats?format=json")
	if nil != err {
		if err == io.EOF {
			return nil
		}
		log.Warnf("[Metric] metric server received stat http error %s", err)
		return nil
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		log.Warnf("[Metric] fail to read all text from stat body stream, error: %s", err)
		return nil
	}
	statsObject := &StatsObject{}
	err = json.Unmarshal(body, statsObject)
	if nil != err {
		log.Warnf("[Metric] fail to unmarshal stat response text %s to cluster, error %s", string(body), err)
		return nil
	}
	return statsObject
}

const clusterRegex = "^cluster\\..+\\.upstream_rq_time$"

func (s *Server) parseUpstreamDelay() map[string]float64 {
	var retValues = make(map[string]float64)
	statsObject := s.getClusterStats()
	if nil == statsObject {
		return retValues
	}
	if nil == statsObject || len(statsObject.Stats) == 0 {
		return retValues
	}
	for _, stat := range statsObject.Stats {
		if nil == stat.Histograms {
			continue
		}
		computedQuantiles := stat.Histograms.ComputedQuantiles
		if len(computedQuantiles) == 0 {
			continue
		}
		for _, computedQuantile := range computedQuantiles {
			matches, err := regexp.Match(clusterRegex, []byte(computedQuantile.Name))
			if nil != err {
				log.Warnf("fail to match regex %s by %s, err: %v", clusterRegex, computedQuantile.Name, err)
				continue
			}
			if !matches {
				continue
			}
			var svcName string = computedQuantile.Name[strings.Index(computedQuantile.Name, ".")+1:]
			svcName = svcName[0:strings.LastIndex(svcName, ".")]
			values := computedQuantile.Values
			if len(values) < 3 {
				continue
			}
			// get the P50 value to perform as average
			value := values[2].Cumulative
			if nil == value {
				continue
			}
			switch v := value.(type) {
			case float64:
				retValues[svcName] = v
			case int:
				retValues[svcName] = float64(v)
			}

		}
	}
	return retValues
}

func (s *Server) reportMetricByCluster(values map[InstanceMetricKey]*InstanceMetricValue) {
	resp, err := http.Get("http://127.0.0.1:15000/clusters?format=json")
	if nil != err {
		if err == io.EOF {
			return
		}
		log.Warnf("[Metric] metric server received clusters http error %s", err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		log.Warnf("[Metric] fail to read all text from body stream, error: %s", err)
		return
	}
	clusters := &adminv3.Clusters{}
	err = jsonpb.UnmarshalString(string(body), clusters)
	if nil != err {
		log.Warnf("[Metric] fail to unmarshal response text %s to cluster, error %s", string(body), err)
		return
	}
	delayValues := s.parseUpstreamDelay()
	log.Infof("[Metric] parsed upstream delay is %v", delayValues)
	clusterStatuses := clusters.GetClusterStatuses()
	if len(clusterStatuses) > 0 {
		for _, clusterStatus := range clusterStatuses {
			clusterName := clusterStatus.GetName()
			hostStatuses := clusterStatus.GetHostStatuses()
			if len(hostStatuses) == 0 {
				continue
			}
			for _, hostStatus := range hostStatuses {
				address := hostStatus.GetAddress()
				if nil == address {
					continue
				}
				socketAddress := address.GetSocketAddress()
				if nil == socketAddress {
					continue
				}
				metricKey := InstanceMetricKey{
					ClusterName: clusterName, Host: socketAddress.GetAddress(), Port: socketAddress.GetPortValue()}
				metricValue := &InstanceMetricValue{}
				stats := hostStatus.GetStats()
				if len(stats) > 0 {
					for _, stat := range stats {
						if stat.GetName() == "rq_total" {
							metricValue.RqTotal = stat.GetValue()
						} else if stat.GetName() == "rq_success" {
							metricValue.RqSuccess = stat.GetValue()
						} else if stat.GetName() == "rq_error" {
							metricValue.RqError = stat.GetValue()
						}
					}
				}
				subMetricValue := &InstanceMetricValue{}
				latestValue, ok := values[metricKey]
				if !ok {
					subMetricValue = metricValue
				} else {
					if metricValue.RqTotal > latestValue.RqTotal {
						subMetricValue.RqTotal = metricValue.RqTotal - latestValue.RqTotal
					}
					if metricValue.RqSuccess > latestValue.RqSuccess {
						subMetricValue.RqSuccess = metricValue.RqSuccess - latestValue.RqSuccess
					}
					if metricValue.RqError > latestValue.RqError {
						subMetricValue.RqError = metricValue.RqError - latestValue.RqError
					}
				}
				values[metricKey] = metricValue
				s.reportMetrics(metricKey, subMetricValue, delayValues[metricKey.ClusterName])
			}
		}
	}
}

func (s *Server) reportMetrics(metricKey InstanceMetricKey, subMetricValue *InstanceMetricValue, delay float64) {
	log.Infof("start to report metric data %s, metric key %s, delay %v", *subMetricValue, metricKey, delay)
	for i := 0; i < int(subMetricValue.RqSuccess); i++ {
		s.reportStatus(metricKey, model.RetSuccess, 200, delay)
	}
	for i := 0; i < int(subMetricValue.RqError); i++ {
		s.reportStatus(metricKey, model.RetFail, 500, delay)
	}
}

func (s *Server) reportStatus(metricKey InstanceMetricKey, retStatus model.RetStatus, code int32, delay float64) {
	callResult := &polaris.ServiceCallResult{}
	callResult.SetRetStatus(retStatus)
	namingInstance := &namingpb.Instance{}
	namingInstance.Service = &wrappers.StringValue{Value: metricKey.ClusterName}
	namingInstance.Namespace = &wrappers.StringValue{Value: s.namespace}
	namingInstance.Host = &wrappers.StringValue{Value: metricKey.Host}
	namingInstance.Port = &wrappers.UInt32Value{Value: metricKey.Port}
	instance := pb.NewInstanceInProto(namingInstance,
		&model.ServiceKey{Namespace: s.namespace, Service: metricKey.ClusterName}, nil)
	callResult.SetCalledInstance(instance)
	callResult.SetRetCode(code)
	callResult.SetDelay(time.Duration(delay) * time.Millisecond)
	if err := s.consumer.UpdateServiceCallResult(callResult); nil != err {
		log.Warnf("[Metric] fail to update service call result for %s, err: %v", metricKey, err)
	}
}
