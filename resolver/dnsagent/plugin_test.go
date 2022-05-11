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

package dnsagent

import (
	"encoding/hex"
	"net"
	"testing"

	"github.com/miekg/dns"
	"github.com/polarismesh/polaris-go/pkg/model"
	"github.com/stretchr/testify/assert"
)

func Test_encodeIPAsFqdn(t *testing.T) {
	type args struct {
		ip     net.IP
		svcKey *model.ServiceKey
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test-ipv4",
			args: args{
				ip: net.ParseIP("127.0.0.1"),
				svcKey: &model.ServiceKey{
					Namespace: "default",
					Service:   "sidecar",
				},
			},
			want: "",
		},
		{
			name: "test-ipv6",
			args: args{
				ip: net.ParseIP("1050:0000:0000:0000:0005:0600:300c:326b"),
				svcKey: &model.ServiceKey{
					Namespace: "default",
					Service:   "sidecar",
				},
			},
			want: "",
		},
		{
			name: "test-ipv6",
			args: args{
				ip: net.ParseIP("1050:0:0:0:5:600:300c:326b"),
				svcKey: &model.ServiceKey{
					Namespace: "default",
					Service:   "sidecar",
				},
			},
			want: "",
		},
		{
			name: "test-ipv6",
			args: args{
				ip: net.ParseIP("ff06::c3"),
				svcKey: &model.ServiceKey{
					Namespace: "default",
					Service:   "sidecar",
				},
			},
			want: "",
		},
		{
			name: "test-ipv6",
			args: args{
				ip: net.ParseIP("::ffff:192.1.56.10"),
				svcKey: &model.ServiceKey{
					Namespace: "default",
					Service:   "sidecar",
				},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encodeIPAsFqdn(tt.args.ip, *tt.args.svcKey)
			t.Log(got)

			ipStr := dns.SplitDomainName(got)[0]

			ret, err := hex.DecodeString(ipStr)
			assert.NoError(t, err)

			t.Log(net.IP(ret))

			assert.Equal(t, tt.args.ip, net.IP(ret))
		})
	}
}
