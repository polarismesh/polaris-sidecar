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

package mesh

import (
	"encoding/json"
	"github.com/polarismesh/polaris-sidecar/log"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

type clusterConfig struct {
	Configs []config `json:"configs"`
}

type config struct {
	Cluster cluster `json:"cluster"`
}

type cluster struct {
	Name string `json:"name"`
}

type registry interface {
	GetCurrentNsService() ([]string, error)
}

func newRegistry(conf *resolverConfig) (registry, error) {
	r := &envoyRegistry{
		conf: conf,
	}
	return r, nil
}

type envoyRegistry struct {
	conf *resolverConfig
}

func (r *envoyRegistry) GetCurrentNsService() ([]string, error) {

	var req *http.Request
	var resp *http.Response
	var err error
	var body []byte
	var services []string
	configDump := &clusterConfig{}

	reqUrl := "http://" + r.conf.RegistryHost + ":" + strconv.Itoa(r.conf.RegistryPort) + "/config_dump"

	if req, err = http.NewRequest(http.MethodGet, reqUrl, nil); err != nil {
		log.Errorf("[Mesh] failed to construct request, %v", err)
		return nil, err
	}

	param := req.URL.Query()
	param.Add("resource", "dynamic_active_clusters")

	req.URL.RawQuery = param.Encode()

	httpClient := &http.Client{
		Timeout: 1 * time.Second,
	}

	if resp, err = httpClient.Do(req); err != nil {
		log.Errorf("[Mesh] fail to request envoy sidecar, %v", err)
		return nil, err
	}

	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		log.Errorf("[Mesh] fail to read response, %v", err)
		return nil, err
	}

	if err = json.Unmarshal(body, configDump); err != nil {
		log.Errorf("[Mesh] fail to unmarshal envoy response, %v", err)
		return nil, err
	}

	for _, c := range configDump.Configs {
		services = append(services, c.Cluster.Name)
	}

	return services, nil
}
