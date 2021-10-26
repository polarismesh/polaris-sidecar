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

package main

import (
	"fmt"
	"git.code.oa.com/polaris/polaris-go/api"
	"git.code.oa.com/polaris/polaris-sidecar/conf"
	"git.code.oa.com/polaris/polaris-sidecar/dnsServer"
	"github.com/kelseyhightower/envconfig"
	"log"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"reflect"
	"sort"
	"syscall"
	"time"
)

var exitSignals = []os.Signal{
	syscall.SIGINT, syscall.SIGTERM, syscall.SIGSEGV,
}

var currentServices []string

// StartDNS start DNS server
func StartDNS(conf *conf.DnsConfig) {
	fmt.Println("start dns server")

	h, err := dnsServer.NewLocalDNSServer()
	if err != nil {
		fmt.Printf("init local dns cache server error , %v \n", err)
		return
	}

	errChan := make(chan error, 1)
	// start reload task
	go startTimingReloadDnsCache(conf, h, errChan)

	// start dns server
	h.StartDNS()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, exitSignals...)

	for {
		select {
		case s := <-ch:
			fmt.Printf("catch singal(%v), stop dns server \n", s)
			return
		case err := <-errChan:
			fmt.Printf("error in start dns server, err %v \n", err)
			return
		}
	}
}

func startTimingReloadDnsCache(conf *conf.DnsConfig, h *dnsServer.LocalDNSServer, errChan chan error) {
	r, err := dnsServer.NewRegistry(conf)
	if err != nil {
		fmt.Printf("init dns server error, %v \n", err)
		errChan <- err
		return
	}

	interval := time.Duration(conf.ReloadDnsCacheInterval) * time.Second

	ticker := time.NewTicker(interval)

	for {
		select {
		case <-ticker.C:
			services, err := r.GetCurrentNsService()
			if err != nil {
				fmt.Printf("error to get services, err %v \n", err)
				continue
			}
			if ifServiceListChanged(services) {
				h.UpdateLookupTable(services, conf.DNSAnswerIp)
				sort.Strings(currentServices)
				currentServices = services
			}
		}
	}
}

func ifServiceListChanged(newNsServices []string) bool {
	sort.Strings(newNsServices)
	if reflect.DeepEqual(currentServices, newNsServices) {
		return false
	}
	return true
}

// main entry
func main() {

	err := api.SetLoggersDir("./log/")
	if err != nil {
		fmt.Println("set loggerDir error, ", err)
		return
	}

	conf := &conf.DnsConfig{}

	if err := envconfig.Process("", conf); err != nil {
		log.Fatal("exit with loading config error. ", err)
	}

	conf.SetDefault()

	StartDNS(conf)
}
