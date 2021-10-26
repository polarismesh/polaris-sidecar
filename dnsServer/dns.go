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
	"strings"
	"sync/atomic"
)

type LocalDNSServer struct {
	// dns look up table
	lookupTable atomic.Value

	udpDNSProxy *dnsProxy
	tcpDNSProxy *dnsProxy

	resolvConfServers []string
	searchNamespaces  []string
}

const (
	defaultTTLInSeconds = 30
)

func (h *LocalDNSServer) UpdateLookupTable(polarisServices []string, dnsResponseIp string) {
	lookupTable := &LookupTable{
		allHosts: map[string]struct{}{},
		name4:    map[string][]dns.RR{},
		name6:    map[string][]dns.RR{},
		cname:    map[string][]dns.RR{},
	}

	var altHosts map[string]struct{}
	for _, service := range polarisServices {
		altHosts = map[string]struct{}{service + ".": {}}

		lookupTable.buildDNSAnswers(altHosts, []net.IP{net.ParseIP(dnsResponseIp)}, nil, h.searchNamespaces)
	}
	h.lookupTable.Store(lookupTable)
	log.GetBaseLogger().Infof("updated lookup table with %d hosts", len(lookupTable.allHosts))
}

func (table *LookupTable) buildDNSAnswers(altHosts map[string]struct{}, ipv4 []net.IP, ipv6 []net.IP, searchNamespaces []string) {
	for h := range altHosts {
		h = strings.ToLower(h)
		table.allHosts[h] = struct{}{}
		if len(ipv4) > 0 {
			table.name4[h] = a(h, ipv4)
		}
		if len(ipv6) > 0 {
			table.name6[h] = aaaa(h, ipv6)
		}
		if len(searchNamespaces) > 0 {
			// NOTE: Right now, rather than storing one expanded host for each one of the search namespace
			// entries, we are going to store just the first one (assuming that most clients will
			// do sequential dns resolution, starting with the first search namespace)

			// host h already ends with a .
			// search namespace might not. So we append one in the end if needed
			expandedHost := strings.ToLower(h + searchNamespaces[0])
			if !strings.HasSuffix(searchNamespaces[0], ".") {
				expandedHost += "."
			}
			// make sure this is not a proper hostname
			// if host is productpage, and search namespace is ns1.svc.cluster.local
			// then the expanded host productpage.ns1.svc.cluster.local is a valid hostname
			// that is likely to be already present in the altHosts
			if _, exists := altHosts[expandedHost]; !exists {
				table.cname[expandedHost] = cname(expandedHost, h)
				table.allHosts[expandedHost] = struct{}{}
			}
		}
	}
}

// Given a host, this function first decides if the host is part of our service registry.
// If it is not part of the registry, return nil so that caller queries upstream. If it is part
// of registry, we will look it up in one of our tables, failing which we will return NXDOMAIN.
func (table *LookupTable) lookupHost(qtype uint16, hostname string) ([]dns.RR, bool) {
	var hostFound bool
	if _, hostFound = table.allHosts[hostname]; !hostFound {
		// this is not from our registry
		return nil, false
	}

	var out []dns.RR
	// Odds are, the first query will always be an expanded hostname
	// (productpage.ns1.svc.cluster.local.ns1.svc.cluster.local)
	// So lookup the cname table first
	cn := table.cname[hostname]
	if len(cn) > 0 {
		// this was a cname match
		hostname = cn[0].(*dns.CNAME).Target
	}
	var ipAnswers []dns.RR
	switch qtype {
	case dns.TypeA:
		ipAnswers = table.name4[hostname]
	case dns.TypeAAAA:
		ipAnswers = table.name6[hostname]
	default:
		// TODO: handle PTR records for reverse dns lookups
		return nil, false
	}

	if len(ipAnswers) > 0 {
		// We will return a chained response. In a chained response, the first entry is the cname record,
		// and the second one is the A/AAAA record itself. Some clients do not follow cname redirects
		// with additional DNS queries. Instead, they expect all the resolved records to be in the same
		// big DNS response (presumably assuming that a recursive DNS query should do the deed, resolve
		// cname et al and return the composite response).
		out = append(out, cn...)
		out = append(out, ipAnswers...)
	}
	return out, hostFound
}

type LookupTable struct {
	// This table will be first looked up to see if the host is something that we got a Nametable entry for
	// (i.e. came from istiod's service registry). If it is, then we will be able to confidently return
	// NXDOMAIN errors for AAAA records for such hosts when only A records exist (or vice versa). If the
	// host does not exist in this map, then we will return nil, causing the caller to query the upstream
	// DNS server to resolve the host. Without this map, we would end up making unnecessary upstream DNS queries
	// for hosts that will never resolve (e.g., AAAA for svc1.ns1.svc.cluster.local.svc.cluster.local.)
	allHosts map[string]struct{}

	// The key is a FQDN matching a DNS query (like example.com.), the value is pre-created DNS RR records
	// of A or AAAA type as appropriate.
	name4 map[string][]dns.RR
	name6 map[string][]dns.RR
	// The cname records here (comprised of different variants of the hosts above,
	// expanded by the search namespaces) pointing to the actual host.
	cname map[string][]dns.RR
}

func NewLocalDNSServer() (*LocalDNSServer, error) {

	h := &LocalDNSServer{}

	dnsConfig, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		log.GetBaseLogger().Errorf("failed to load /etc/resolv.conf: %v", err)
		return nil, err
	}

	if dnsConfig != nil {
		for _, s := range dnsConfig.Servers {
			h.resolvConfServers = append(h.resolvConfServers, net.JoinHostPort(s, "53"))
		}
		h.searchNamespaces = dnsConfig.Search
	}

	log.GetBaseLogger().Infof("upstream servers: %v", h.resolvConfServers)

	if h.udpDNSProxy, err = newDNSProxy("udp", h); err != nil {
		return nil, err
	}
	if h.tcpDNSProxy, err = newDNSProxy("tcp", h); err != nil {
		return nil, err
	}

	return h, nil
}

func (h *LocalDNSServer) StartDNS() {
	go h.udpDNSProxy.start()
	go h.tcpDNSProxy.start()
}

func (h *LocalDNSServer) ServeDNS(proxy *dnsProxy, w dns.ResponseWriter, req *dns.Msg) {
	var response *dns.Msg
	log.GetBaseLogger().Infof("protocol", proxy.protocol, "edns", req.IsEdns0() != nil)

	log.GetBaseLogger().Debugf("request %v", req)

	if len(req.Question) == 0 {
		response = new(dns.Msg)
		response.SetReply(req)
		response.Rcode = dns.RcodeServerFailure
		_ = w.WriteMsg(response)
		return
	}

	lp := h.lookupTable.Load()
	if lp == nil {
		response = new(dns.Msg)
		response.SetReply(req)
		response.Rcode = dns.RcodeServerFailure
		log.GetBaseLogger().Debugf("dns request before lookup table is loaded")
		_ = w.WriteMsg(response)
		return
	}

	lookupTable := lp.(*LookupTable)
	var answers []dns.RR

	hostname := strings.ToLower(req.Question[0].Name)
	answers, hostFound := lookupTable.lookupHost(req.Question[0].Qtype, hostname)

	if hostFound {
		response = new(dns.Msg)
		response.SetReply(req)
		response.Authoritative = true
		response.Answer = answers
		response.Truncate(size(proxy.protocol, req))
		log.GetBaseLogger().Debugf("response for hostname %q (found=true): %v", hostname, response)
		_ = w.WriteMsg(response)
		return
	}

	response = h.queryUpstream(proxy.upstreamClient, req)
	response.Truncate(size(proxy.protocol, req))
	log.GetBaseLogger().Debugf("response for hostname %s (found=false): %v", hostname, response)
	_ = w.WriteMsg(response)
}

// Borrowed from https://github.com/coredns/coredns/blob/master/plugin/hosts/hosts.go
// a takes a slice of net.IPs and returns a slice of A RRs.
func a(host string, ips []net.IP) []dns.RR {
	answers := make([]dns.RR, len(ips))
	for i, ip := range ips {
		r := new(dns.A)
		r.Hdr = dns.RR_Header{Name: host, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: defaultTTLInSeconds}
		r.A = ip
		answers[i] = r
	}
	return answers
}

// aaaa takes a slice of net.IPs and returns a slice of AAAA RRs.
func aaaa(host string, ips []net.IP) []dns.RR {
	answers := make([]dns.RR, len(ips))
	for i, ip := range ips {
		r := new(dns.AAAA)
		r.Hdr = dns.RR_Header{Name: host, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: defaultTTLInSeconds}
		r.AAAA = ip
		answers[i] = r
	}
	return answers
}

func cname(host string, targetHost string) []dns.RR {
	answer := new(dns.CNAME)
	answer.Hdr = dns.RR_Header{
		Name:   host,
		Rrtype: dns.TypeCNAME,
		Class:  dns.ClassINET,
		Ttl:    defaultTTLInSeconds,
	}
	answer.Target = targetHost
	return []dns.RR{answer}
}

// Size returns if buffer size *advertised* in the requests OPT record.
// Or when the request was over TCP, we return the maximum allowed size of 64K.
func size(proto string, r *dns.Msg) int {
	size := uint16(0)
	if o := r.IsEdns0(); o != nil {
		size = o.UDPSize()
	}

	// normalize size
	size = ednsSize(proto, size)
	return int(size)
}

// ednsSize returns a normalized size based on proto.
func ednsSize(proto string, size uint16) uint16 {
	if proto == "tcp" {
		return dns.MaxMsgSize
	}
	if size < dns.MinMsgSize {
		return dns.MinMsgSize
	}
	return size
}

// TODO: Figure out how to send parallel queries to all nameservers
func (h *LocalDNSServer) queryUpstream(upstreamClient *dns.Client, req *dns.Msg) *dns.Msg {
	var response *dns.Msg
	for _, upstream := range h.resolvConfServers {
		cResponse, _, err := upstreamClient.Exchange(req, upstream)
		if err == nil {
			response = cResponse
			break
		} else {
			log.GetBaseLogger().Infof("upstream failure: %v", err)
		}
	}
	if response == nil {
		response = new(dns.Msg)
		response.SetReply(req)
		response.Rcode = dns.RcodeServerFailure
	}
	return response
}
