package bootstrap

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"

	"github.com/polarismesh/polaris-sidecar/log"
	"github.com/polarismesh/polaris-sidecar/resolver"
)

// SidecarConfig global sidecar config struct
type SidecarConfig struct {
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
	TimeoutSec  int      `yaml:"timeoutSec"`
	NameServers []string `yaml:"name_servers"`
}

// 设置关键默认值
func defaultSidecarConfig() *SidecarConfig {
	return &SidecarConfig{
		Port: 53,
		Recurse: &RecurseConfig{
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
				Name:   "discovery",
				DnsTtl: 0,
				Enable: true,
			},
			{
				Name:   "mesh",
				DnsTtl: 120,
				Enable: false,
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

func isFile(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !s.IsDir()
}

// parseYamlConfig parse config file to object
func parseYamlConfig(configFile string) (*SidecarConfig, error) {
	sidecarConfig := defaultSidecarConfig()
	if !isFile(configFile) {
		return sidecarConfig, nil
	}
	buf, err := ioutil.ReadFile(configFile)
	if nil != err {
		return nil, errors.New(fmt.Sprintf("read file %s error", configFile))
	}
	decoder := yaml.NewDecoder(bytes.NewBuffer(buf))
	if err = decoder.Decode(sidecarConfig); nil != err {
		return nil, errors.New(fmt.Sprintf("parse yaml %s error:%s", configFile, err.Error()))
	}
	return sidecarConfig, nil
}
