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

package client

import (
	"errors"
	"sync"

	"github.com/polarismesh/polaris-go"
	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/config"
	"github.com/polarismesh/polaris-go/plugin/statreporter/prometheus"
)

var (
	lock       sync.Mutex
	SDKContext api.SDKContext
)

func InitSDKContext(conf *Config) error {
	lock.Lock()
	defer lock.Unlock()
	if SDKContext != nil {
		return nil
	}
	sdkCfg := config.NewDefaultConfiguration(conf.Addresses)
	sdkCfg.Consumer.CircuitBreaker.SetEnable(false)
	if conf.Metrics != nil {
		sdkCfg.Global.StatReporter.SetEnable(true)
		sdkCfg.Global.StatReporter.SetChain([]string{"prometheus"})
		sdkCfg.Global.StatReporter.SetPluginConfig("prometheus", &prometheus.Config{
			
		})
	}
	sdkCtx, err := polaris.NewSDKContextByConfig(sdkCfg)
	if err != nil {
		return err
	}
	SDKContext = sdkCtx
	return nil
}

func GetConsumerAPI() (polaris.ConsumerAPI, error) {
	if SDKContext == nil {
		return nil, errors.New("polaris SDKContext is nil")
	}
	return polaris.NewConsumerAPIByContext(SDKContext), nil
}

func GetProviderAPI() (polaris.ProviderAPI, error) {
	if SDKContext == nil {
		return nil, errors.New("polaris SDKContext is nil")
	}
	return polaris.NewProviderAPIByContext(SDKContext), nil
}

func GetLimitAPI() (polaris.LimitAPI, error) {
	if SDKContext == nil {
		return nil, errors.New("polaris SDKContext is nil")
	}
	return polaris.NewLimitAPIByContext(SDKContext), nil
}
