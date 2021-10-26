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

package dnsServer

import (
	"git.code.oa.com/polaris/polaris-go/pkg/log"
	"github.com/miekg/dns"
	"net"
)

type dnsProxy struct {
	downstreamMux    *dns.ServeMux
	downstreamServer *dns.Server

	upstreamClient *dns.Client
	protocol       string
	resolver       *LocalDNSServer
}

func newDNSProxy(protocol string, resolver *LocalDNSServer) (*dnsProxy, error) {
	p := &dnsProxy{
		downstreamMux:    dns.NewServeMux(),
		downstreamServer: &dns.Server{},
		upstreamClient: &dns.Client{
			Net: protocol,
		},
		protocol: protocol,
		resolver: resolver,
	}

	var err error
	p.downstreamMux.Handle(".", p)
	p.downstreamServer.Handler = p.downstreamMux
	if protocol == "udp" {
		p.downstreamServer.PacketConn, err = net.ListenPacket("udp", "localhost:15053")
	} else {
		p.downstreamServer.Listener, err = net.Listen("tcp", "localhost:15053")
	}
	if err != nil {
		log.GetBaseLogger().Errorf("Failed to listen on %s port 15053 %v", protocol, err)
		return nil, err
	}
	return p, nil
}

func (d *dnsProxy) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	d.resolver.ServeDNS(d, w, r)
}

func (d *dnsProxy) start() {
	log.GetBaseLogger().Infof("Starting local %s DNS server at localhost:15053", d.protocol)
	err := d.downstreamServer.ActivateAndServe()
	if err != nil {
		log.GetBaseLogger().Errorf("Local %s DNS server terminated: %v", err)
	}
}
