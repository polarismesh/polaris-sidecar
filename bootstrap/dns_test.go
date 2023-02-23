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
	"testing"
)

func Test_dnsHandler_preprocess(t *testing.T) {
	type fields struct {
		searchNames []string
	}
	type args struct {
		qname string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "",
			fields: fields{
				searchNames: []string{"polaris-system.svc.cluster.local.", "svc.cluster.local.", "cluster.local."},
			},
			args: args{
				qname: "polaris.polaris-system.polaris-system.svc.cluster.local.",
			},
			want: "polaris.polaris-system.",
		},
		{
			name: "",
			fields: fields{
				searchNames: []string{"polaris-system.svc.cluster.local.", "svc.cluster.local.", "cluster.local."},
			},
			args: args{
				qname: "polaris.polaris-system.",
			},
			want: "polaris.polaris-system.",
		},
		{
			name: "",
			fields: fields{
				searchNames: []string{"polaris-system.svc.cluster.local.", "svc.cluster.local.", "cluster.local."},
			},
			args: args{
				qname: "polaris.polaris-system.svc.cluster.local.",
			},
			want: "polaris.",
		},
		{
			name: "",
			fields: fields{
				searchNames: []string{"svc.cluster.local.", "polaris-system.svc.cluster.local.", "cluster.local."},
			},
			args: args{
				qname: "polaris.polaris-system.svc.cluster.local.",
			},
			want: "polaris.polaris-system.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &dnsHandler{
				searchNames: tt.fields.searchNames,
			}
			if got := d.preprocess(tt.args.qname); got != tt.want {
				t.Errorf("dnsHandler.preprocess() = %v, want %v", got, tt.want)
			}
		})
	}
}
