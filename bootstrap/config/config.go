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

package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/hashicorp/go-multierror"
	sdkconf "github.com/polarismesh/polaris-go/pkg/config"
	"gopkg.in/yaml.v2"

	"github.com/polarismesh/polaris-sidecar/envoy/metrics"
	"github.com/polarismesh/polaris-sidecar/envoy/rls"
	"github.com/polarismesh/polaris-sidecar/pkg/log"
	"github.com/polarismesh/polaris-sidecar/resolver"
)

const defaultSvcSuffix = "."

// BootConfig simple config for bootstrap
type BootConfig struct {
	Bind                        string
	Port                        int
	LogLevel                    string
	RecurseEnabled              string
	ResolverDnsAgentEnabled     string
	ResolverDnsAgentRouteLabels string
	ResolverMeshProxyEnabled    string
}

// SidecarConfig global sidecar config struct
type SidecarConfig struct {
	PolarisConfig *PolarisConfig          `yaml:"polaris"`
	Bind          string                  `yaml:"bind"`
	Port          int                     `yaml:"port"`
	Namespace     string                  `yaml:"namespace"`
	MTLS          *MTLSConfiguration      `yaml:"mtls"`
	Logger        *log.Options            `yaml:"logger"`
	Recurse       *resolver.RecurseConfig `yaml:"recurse"`
	Resolvers     []*resolver.ConfigEntry `yaml:"resolvers"`
	Metrics       *metrics.MetricConfig   `yaml:"metrics"`
	RateLimit     *rls.Config             `yaml:"ratelimit"`
	Debugger      *DebugConfig            `yaml:"debugger"`
}

type PolarisConfig struct {
	Adddresses []string                    `yaml:"adddresses"`
	Location   *sdkconf.LocationConfigImpl `yaml:"location"`
}

type Location struct {
	Providers []*LocationProvider `yaml:"providers"`
}

type LocationProvider struct {
	Type    string                 `yaml:"type" json:"type"`
	Options map[string]interface{} `yaml:"options" json:"options"`
}

type MTLSConfiguration struct {
	Enable   bool   `yaml:"enable"`
	CAServer string `yaml:"ca_server"`
}

type DebugConfig struct {
	Enable bool  `yaml:"enable"`
	Port   int32 `yaml:"port"`
}

// String toString output
func (s SidecarConfig) String() string {
	strBytes, err := yaml.Marshal(&s)
	if nil != err {
		return ""
	}
	return string(strBytes)
}

// 设置关键默认值
func defaultSidecarConfig() *SidecarConfig {
	return &SidecarConfig{
		PolarisConfig: &PolarisConfig{
			Adddresses: []string{
				"127.0.0.1:8091",
			},
		},
		Bind: "0.0.0.0",
		Port: 53,
		Recurse: &resolver.RecurseConfig{
			Enable:     false,
			TimeoutSec: 1,
		},
		MTLS: &MTLSConfiguration{
			Enable: false,
		},
		Logger: &log.Options{
			OutputPaths: []string{
				"stdout",
			},
			ErrorOutputPaths: []string{
				"stderr",
			},
			RotateOutputPath:      "log/polaris-sidecar.log",
			ErrorRotateOutputPath: "log/polaris-sidecar-error.log",
			RotationMaxAge:        7,
			RotationMaxBackups:    100,
			RotationMaxSize:       100,
			OutputLevel:           "info",
		},
		Resolvers: []*resolver.ConfigEntry{
			{
				Name:   resolver.PluginNameDnsAgent,
				DnsTtl: 10,
				Enable: true,
				Suffix: defaultSvcSuffix,
			},
			{
				Name:   resolver.PluginNameMeshProxy,
				DnsTtl: 120,
				Enable: false,
				Suffix: defaultSvcSuffix,
				Option: map[string]interface{}{
					"reload_interval_sec": 30,
					"dns_answer_ip":       "10.4.4.4",
				},
			},
		},
		RateLimit: &rls.Config{
			Enable:  false,
			Network: "unix",
			Address: rls.DefaultRLSAddress,
		},
		Metrics: &metrics.MetricConfig{
			Enable: false,
			Port:   15985,
		},
		Debugger: &DebugConfig{
			Enable: true,
			Port:   50000,
		},
	}
}

func (s *SidecarConfig) BindLocalhost() bool {
	bindIP := net.ParseIP(s.Bind)
	return bindIP.IsLoopback() || bindIP.IsUnspecified()
}

func (s *SidecarConfig) verify() error {
	var errs multierror.Error
	if len(s.Bind) == 0 {
		errs.Errors = append(errs.Errors, errors.New("host should not empty"))
	}
	if s.Port <= 0 {
		errs.Errors = append(errs.Errors, errors.New("port should greater than 0"))
	}
	if s.Recurse.TimeoutSec <= 0 {
		errs.Errors = append(errs.Errors, errors.New("recurse.timeout should greater than 0"))
	}
	if len(s.Resolvers) == 0 {
		errs.Errors = append(errs.Errors, errors.New("you should at least config one resolver"))
	}
	var hasOneEnable bool
	for idx, resolverConfig := range s.Resolvers {
		if len(resolverConfig.Name) == 0 {
			errs.Errors = append(errs.Errors, errors.New(fmt.Sprintf("resolver %d config name is empty", idx)))
		}
		if resolverConfig.DnsTtl < 0 {
			errs.Errors = append(errs.Errors, errors.New(
				fmt.Sprintf("resolver %d config dnsttl should greater or equals to 0", idx)))
		}
		if resolverConfig.Enable {
			hasOneEnable = true
		}
	}
	if !hasOneEnable {
		errs.Errors = append(errs.Errors, errors.New("you should at least enable one resolver"))
	}
	return errs.ErrorOrNil()
}

const (
	labelSep = ","
	kvSep    = ":"
)

func parseLabels(labels string) map[string]string {
	if len(labels) == 0 {
		return nil
	}
	values := make(map[string]string)
	tokens := strings.Split(labels, labelSep)
	for _, token := range tokens {
		if len(token) == 0 {
			continue
		}
		pairs := strings.Split(token, kvSep)
		if len(pairs) > 1 {
			values[pairs[0]] = pairs[1]
		}
	}
	return values
}

func (s *SidecarConfig) mergeEnv() {
	s.Bind = getEnvStringValue(EnvSidecarBind, s.Bind)
	s.Port = getEnvIntValue(EnvSidecarPort, s.Port)
	s.Namespace = getEnvStringValue(EnvSidecarNamespace, s.Namespace)
	s.PolarisConfig.Adddresses = getEnvStringsValue(EnvPolarisAddress, s.PolarisConfig.Adddresses)
	s.MTLS.Enable = getEnvBoolValue(EnvSidecarMtlsEnable, s.MTLS.Enable)
	s.MTLS.CAServer = getEnvStringValue(EnvSidecarMtlsCAServer, s.MTLS.CAServer)
	s.RateLimit.Enable = getEnvBoolValue(EnvSidecarRLSEnable, s.MTLS.Enable)
	s.Recurse.Enable = getEnvBoolValue(EnvSidecarRecurseEnable, s.Recurse.Enable)
	s.Recurse.TimeoutSec = getEnvIntValue(EnvSidecarRecurseTimeout, s.Recurse.TimeoutSec)
	s.Logger.RotateOutputPath = getEnvStringValue(EnvSidecarLogRotateOutputPath, s.Logger.RotateOutputPath)
	s.Logger.ErrorRotateOutputPath = getEnvStringValue(EnvSidecarLogErrorRotateOutputPath, s.Logger.ErrorRotateOutputPath)
	s.Logger.RotationMaxSize = getEnvIntValue(EnvSidecarLogRotationMaxSize, s.Logger.RotationMaxSize)
	s.Logger.RotationMaxBackups = getEnvIntValue(EnvSidecarLogRotationMaxBackups, s.Logger.RotationMaxBackups)
	s.Logger.RotationMaxAge = getEnvIntValue(EnvSidecarLogRotationMaxAge, s.Logger.RotationMaxAge)
	s.Logger.OutputLevel = getEnvStringValue(EnvSidecarLogLevel, s.Logger.OutputLevel)
	if len(s.Resolvers) > 0 {
		for _, resolverConf := range s.Resolvers {
			resolverConf.Namespace = s.Namespace
			if resolverConf.Name == resolver.PluginNameDnsAgent {
				resolverConf.DnsTtl = getEnvIntValue(EnvSidecarDnsTtl, resolverConf.DnsTtl)
				resolverConf.Enable = getEnvBoolValue(EnvSidecarDnsEnable, resolverConf.Enable)
				resolverConf.Suffix = getEnvStringValue(EnvSidecarDnsSuffix, resolverConf.Suffix)
				routeLabels := getEnvStringValue(EnvSidecarDnsRouteLabels, "")
				if len(routeLabels) > 0 {
					resolverConf.Option = make(map[string]interface{})
					resolverConf.Option["route_labels"] = routeLabels
				}
			} else if resolverConf.Name == resolver.PluginNameMeshProxy {
				resolverConf.DnsTtl = getEnvIntValue(EnvSidecarMeshTtl, resolverConf.DnsTtl)
				resolverConf.Enable = getEnvBoolValue(EnvSidecarMeshEnable, resolverConf.Enable)
				reloadIntervalSec := getEnvIntValue(EnvSidecarMeshReloadInterval, 0)
				if reloadIntervalSec > 0 {
					resolverConf.Option["reload_interval_sec"] = reloadIntervalSec
				}
				dnsAnswerIP := getEnvStringValue(EnvSidecarMeshAnswerIp, "")
				if len(dnsAnswerIP) > 0 {
					resolverConf.Option["dns_answer_ip"] = dnsAnswerIP
				}
			}
		}
	}
	s.Metrics.Enable = getEnvBoolValue(EnvSidecarMetricEnable, s.Metrics.Enable)
	s.Metrics.Port = getEnvIntValue(EnvSidecarMetricListenPort, s.Metrics.Port)
}

func (s *SidecarConfig) mergeBootConfig(config *BootConfig) error {
	var errs multierror.Error
	var err error
	if len(config.Bind) > 0 {
		s.Bind = config.Bind
	}
	if config.Port > 0 {
		s.Port = config.Port
	}
	if len(config.LogLevel) > 0 {
		s.Logger.OutputLevel = config.LogLevel
	}
	if len(config.RecurseEnabled) > 0 {
		s.Recurse.Enable, err = strconv.ParseBool(config.RecurseEnabled)
		if nil != err {
			errs.Errors = append(errs.Errors,
				fmt.Errorf("fail to parse recurse-enabled value to boolean, err: %v", err))
		}
	}
	s.Logger.OutputLevel = config.LogLevel
	if len(config.ResolverDnsAgentEnabled) > 0 || len(config.ResolverDnsAgentRouteLabels) > 0 {
		for _, resolverConfig := range s.Resolvers {
			if resolverConfig.Name == resolver.PluginNameDnsAgent {
				if len(config.ResolverDnsAgentEnabled) > 0 {
					resolverConfig.Enable, err = strconv.ParseBool(config.ResolverDnsAgentEnabled)
					if nil != err {
						errs.Errors = append(errs.Errors,
							fmt.Errorf("fail to parse resolver-dnsAgent-enabled value to boolean, err: %v", err))
					}
				}
				if len(config.ResolverDnsAgentRouteLabels) > 0 {
					labels := parseLabels(config.ResolverDnsAgentRouteLabels)
					if len(labels) > 0 {
						if len(resolverConfig.Option) == 0 {
							resolverConfig.Option = make(map[string]interface{})
						}
						resolverConfig.Option["route_labels"] = labels
					}
				}
				continue
			}
			if resolverConfig.Name == resolver.PluginNameMeshProxy {
				if len(config.ResolverMeshProxyEnabled) > 0 {
					resolverConfig.Enable, err = strconv.ParseBool(config.ResolverMeshProxyEnabled)
					if nil != err {
						errs.Errors = append(errs.Errors,
							fmt.Errorf("fail to parse resolver-meshproxy-enabled value to boolean, err: %v", err))
					}
				}
			}
		}
	}
	return errs.ErrorOrNil()
}

func IsFile(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !s.IsDir()
}

// ParseYamlConfig parse config file to object
func ParseYamlConfig(configFile string, bootConfig *BootConfig) (*SidecarConfig, error) {
	sidecarConfig := defaultSidecarConfig()
	if len(configFile) > 0 && IsFile(configFile) {
		buf, err := ioutil.ReadFile(configFile)
		if nil != err {
			return nil, errors.New(fmt.Sprintf("read file %s error", configFile))
		}
		err = parseYamlContent(buf, sidecarConfig)
		if nil != err {
			return nil, err
		}
	} else {
		log.Warnf("[agent] config file %s not exists, use default sidecar config", configFile)
	}
	sidecarConfig.mergeEnv()
	if err := sidecarConfig.mergeBootConfig(bootConfig); nil != err {
		return nil, err
	}
	return sidecarConfig, sidecarConfig.verify()
}

func parseYamlContent(content []byte, sidecarConfig *SidecarConfig) error {
	data := []byte(os.ExpandEnv(string(content)))
	decoder := yaml.NewDecoder(bytes.NewBuffer(data))
	if err := decoder.Decode(sidecarConfig); nil != err {
		return errors.New(fmt.Sprintf("parse yaml %s error:%s", content, err.Error()))
	}
	log.Infof("[sidecar] config content : %s", content)
	return nil
}
