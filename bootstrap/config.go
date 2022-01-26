package bootstrap

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/polarismesh/polaris-sidecar/log"
	"github.com/polarismesh/polaris-sidecar/resolver"
)

const defaultSvcSuffix = "svc.polaris"

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
	Bind      string                  `yaml:"bind"`
	Port      int                     `yaml:"port"`
	Recurse   *RecurseConfig          `yaml:"recurse"`
	Logger    *log.Options            `yaml:"logger"`
	Resolvers []*resolver.ConfigEntry `yaml:"resolvers"`
}

// String toString output
func (s SidecarConfig) String() string {
	strBytes, err := yaml.Marshal(&s)
	if nil != err {
		return ""
	}
	return string(strBytes)
}

// RecurseConfig recursor name resolve config
type RecurseConfig struct {
	Enable      bool     `yaml:"enable"`
	TimeoutSec  int      `yaml:"timeoutSec"`
	NameServers []string `yaml:"name_servers"`
}

// 设置关键默认值
func defaultSidecarConfig() *SidecarConfig {
	return &SidecarConfig{
		Bind: "0.0.0.0",
		Port: 53,
		Recurse: &RecurseConfig{
			Enable:     false,
			TimeoutSec: 1,
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
				Enable: false,
				Suffix: defaultSvcSuffix,
			},
			{
				Name:   resolver.PluginNameMeshProxy,
				DnsTtl: 120,
				Enable: true,
				Option: map[string]interface{}{
					"registry_host":       "127.0.0.1",
					"registry_port":       15000,
					"reload_interval_sec": 2,
					"dns_answer_ip":       "10.4.4.4",
				},
			},
		},
	}
}

func (s *SidecarConfig) bindLocalhost() bool {
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

func (s *SidecarConfig) merge(config *BootConfig) error {
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

func isFile(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !s.IsDir()
}

// parseYamlConfig parse config file to object
func parseYamlConfig(configFile string, bootConfig *BootConfig) (*SidecarConfig, error) {
	sidecarConfig := defaultSidecarConfig()
	if isFile(configFile) {
		buf, err := ioutil.ReadFile(configFile)
		if nil != err {
			return nil, errors.New(fmt.Sprintf("read file %s error", configFile))
		}
		decoder := yaml.NewDecoder(bytes.NewBuffer(buf))
		if err = decoder.Decode(sidecarConfig); nil != err {
			return nil, errors.New(fmt.Sprintf("parse yaml %s error:%s", configFile, err.Error()))
		}
	}
	err := sidecarConfig.merge(bootConfig)
	if nil != err {
		return nil, err
	}
	return sidecarConfig, sidecarConfig.verify()
}
