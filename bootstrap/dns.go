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
	"context"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"

	"github.com/polarismesh/polaris-sidecar/log"
	"github.com/polarismesh/polaris-sidecar/resolver"
)

type dnsHandler struct {
	protocol        string
	resolvers       []resolver.NamingResolver
	searchNames     []string
	recursorTimeout time.Duration
	recursors       []string
	recurseEnable   bool
}

func (d *dnsHandler) preprocess(qname string) string {
	if len(d.searchNames) == 0 {
		return qname
	}
	for _, searchName := range d.searchNames {
		if strings.HasSuffix(qname, searchName) {
			return qname[:len(qname)-len(searchName)]
		}
	}
	return qname
}

func (d *dnsHandler) sendDnsCode(w dns.ResponseWriter, r *dns.Msg, code int) {
	msg := &dns.Msg{}
	msg.SetReply(r)
	msg.RecursionDesired = true
	msg.RecursionAvailable = true
	msg.Rcode = code
	msg.Truncate(size(d.protocol, r))
	if edns := r.IsEdns0(); edns != nil {
		setEDNS(r, msg, true)
	}
	err := w.WriteMsg(msg)
	if nil != err {
		log.Errorf("[agent] fail to write dns response message, err: %v", err)
	}
}

func (d *dnsHandler) sendDnsResponse(w dns.ResponseWriter, r *dns.Msg, msg *dns.Msg) {
	msg.SetReply(r)
	msg.Truncate(size(d.protocol, r))
	if edns := r.IsEdns0(); edns != nil {
		setEDNS(r, msg, true)
	}
	err := w.WriteMsg(msg)
	if nil != err {
		log.Errorf("[agent] fail to write dns response message, err: %v", err)
	}
}

// ServeDNS handler callback
func (d *dnsHandler) ServeDNS(w dns.ResponseWriter, req *dns.Msg) {
	// questions length is 0, send refused
	if len(req.Question) == 0 {
		d.sendDnsCode(w, req, dns.RcodeRefused)
	}
	// questions type we only accept
	question := req.Question[0]
	question.Name = d.preprocess(question.Name)

	ctx := context.WithValue(context.Background(), resolver.ContextProtocol, d.protocol)
	var resp *dns.Msg
	for _, handler := range d.resolvers {
		resp = handler.ServeDNS(ctx, question)
		if nil != resp {
			d.sendDnsResponse(w, req, resp)
			return
		}
	}
	d.handleRecurse(w, req)
}

// handleRecurse is used to handle recursive DNS queries
func (d *dnsHandler) handleRecurse(resp dns.ResponseWriter, req *dns.Msg) {
	q := req.Question[0]
	network := "udp"
	defer func(s time.Time) {
		log.Debugf("[agent] request served from client, "+
			"question: %s, network: %s, latency: %s, client: %s, client_network: %s",
			q.String(), network, time.Since(s).String(), resp.RemoteAddr().String(), resp.RemoteAddr().Network())
	}(time.Now())

	// Switch to TCP if the client is
	if _, ok := resp.RemoteAddr().(*net.TCPAddr); ok {
		network = "tcp"
	}
	if d.recurseEnable {
		// Recursively resolve
		c := &dns.Client{Net: network, Timeout: d.recursorTimeout}
		var r *dns.Msg
		var rtt time.Duration
		var err error
		for _, recursor := range d.recursors {
			r, rtt, err = c.Exchange(req, recursor)
			// Check if the response is valid and has the desired Response code
			if r != nil && (r.Rcode != dns.RcodeSuccess && r.Rcode != dns.RcodeNameError) {
				log.Warnf("[agent] recurse failed for question, question: %s, rtt: %s, recursor: %s, rcode: %s",
					q.String(), rtt, recursor, dns.RcodeToString[r.Rcode])
				// If we still have recursors to forward the query to,
				// we move forward onto the next one else the loop ends
				continue
			} else if err == nil || (r != nil && r.Truncated) {
				// Forward the response
				log.Infof("[agent] recurse succeeded for question, question: %s, rtt: %s, recursor: %s",
					q.String(), rtt, recursor)
				if err := resp.WriteMsg(r); err != nil {
					log.Warnf("failed to respond, error: %v", err)
				}
				return
			}
			log.Errorf("[agent] recurse failed, error: %v", err)
		}

		// If all resolvers fail, return a SERVFAIL message
		log.Errorf(
			"[agent] all resolvers failed for question from client, question: %s, client: %s, client_network: %s",
			q.String(), resp.RemoteAddr().String(), resp.RemoteAddr().Network())
	}
	d.sendDnsCode(resp, req, dns.RcodeServerFailure)
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

func ednsSubnetForRequest(req *dns.Msg) *dns.EDNS0_SUBNET {
	// IsEdns0 returns the EDNS RR if present or nil otherwise
	edns := req.IsEdns0()

	if edns == nil {
		return nil
	}

	for _, o := range edns.Option {
		if subnet, ok := o.(*dns.EDNS0_SUBNET); ok {
			return subnet
		}
	}

	return nil
}

// setEDNS is used to set the responses EDNS size headers and
// possibly the ECS headers as well if they were present in the
// original request
func setEDNS(request *dns.Msg, response *dns.Msg, ecsGlobal bool) {
	edns := request.IsEdns0()
	if edns == nil {
		return
	}

	// cannot just use the SetEdns0 function as we need to embed
	// the ECS option as well
	ednsResp := new(dns.OPT)
	ednsResp.Hdr.Name = "."
	ednsResp.Hdr.Rrtype = dns.TypeOPT
	ednsResp.SetUDPSize(edns.UDPSize())

	// Setup the ECS option if present
	if subnet := ednsSubnetForRequest(request); subnet != nil {
		subOp := new(dns.EDNS0_SUBNET)
		subOp.Code = dns.EDNS0SUBNET
		subOp.Family = subnet.Family
		subOp.Address = subnet.Address
		subOp.SourceNetmask = subnet.SourceNetmask
		if c := response.Rcode; ecsGlobal || c == dns.RcodeNameError || c == dns.RcodeServerFailure ||
			c == dns.RcodeRefused || c == dns.RcodeNotImplemented {
			// reply is globally valid and should be cached accordingly
			subOp.SourceScope = 0
		} else {
			// reply is only valid for the subnet it was queried with
			subOp.SourceScope = subnet.SourceNetmask
		}
		ednsResp.Option = append(ednsResp.Option, subOp)
	}

	response.Extra = append(response.Extra, ednsResp)
}
