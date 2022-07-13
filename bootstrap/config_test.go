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

package bootstrap

import (
	"fmt"
	"os"
	"testing"
)

func TestParseLabels(t *testing.T) {
	labels := "xx:yy,xx1:yy1,xx2:yy2"
	var values map[string]string
	values = parseLabels(labels)
	fmt.Printf("values are %v\n", values)
}

const testCfg = "resolvers:\n  " +
	"- name: dnsagent\n    " +
	"dns_ttl: 10\n    " +
	"enable: true\n    " +
	"suffix: \".\"\n  " +
	"- name: meshproxy\n   " +
	" dns_ttl: 120\n    " +
	"enable: false\n    " +
	"option:\n      " +
	"registry_host: 127.0.0.1\n      " +
	"registry_port: 15000\n      " +
	"reload_interval_sec: 2\n      " +
	"dns_answer_ip: ${aswip}"

const testAnswerIP = "127.0.0.8"

func TestParseYamlConfig(t *testing.T) {
	err := os.Setenv("aswip", testAnswerIP)
	if nil != err {
		t.Fatal(err)
	}
	cfg := &SidecarConfig{}
	err = parseYamlContent(testCfg, cfg)
	if nil != err {
		t.Fatal(err)
	}
	result := cfg.Resolvers[1].Option["dns_answer_ip"]
	if result != testAnswerIP {
		t.Fatal("answer ip should be " + testAnswerIP)
	}

}
