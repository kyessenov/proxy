// Copyright 2019 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"fmt"
	"testing"
	"time"

	"istio.io/proxy/test/envoye2e/env"
)

const ClientBootstrap = `
node:
  id: client
  cluster: test-cluster
  metadata: { {{ .Vars.ClientMetadata }} }
admin:
  access_log_path: /dev/null
  address:
    socket_address:
      address: 127.0.0.1
      port_value: {{ .Vars.ClientAdmin }}
dynamic_resources:
  ads_config:
    api_type: GRPC
    grpc_services:
    - envoy_grpc:
        cluster_name: xds_cluster
  cds_config:
    ads: {}
  lds_config:
    ads: {}
static_resources:
  clusters:
  - connect_timeout: 1s
    hosts:
    - socket_address:
        address: 127.0.0.1
        port_value: {{ .XDS }}
    http2_protocol_options: {}
    name: xds_cluster
  - name: server
    connect_timeout: 1s
    type: STATIC
    load_assignment:
      cluster_name: server
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: 127.0.0.1
                port_value: {{ .Vars.ServerPort }}
`

const ServerBootstrap = `
node:
  id: server
  cluster: test-cluster
  metadata: { {{ .Vars.ServerMetadata }} }
admin:
  access_log_path: /dev/null
  address:
    socket_address:
      address: 127.0.0.1
      port_value: {{ .Vars.ServerAdmin }}
dynamic_resources:
  ads_config:
    api_type: GRPC
    grpc_services:
    - envoy_grpc:
        cluster_name: xds_cluster
  cds_config:
    ads: {}
  lds_config:
    ads: {}
static_resources:
  clusters:
  - connect_timeout: 1s
    hosts:
    - socket_address:
        address: 127.0.0.1
        port_value: {{ .XDS }}
    http2_protocol_options: {}
    name: xds_cluster
  - name: inbound|9080|http|server.default.svc.cluster.local
    connect_timeout: 1s
    type: STATIC
    load_assignment:
      cluster_name: inbound|9080|http|server.default.svc.cluster.local
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: 127.0.0.1
                port_value: {{ .Vars.BackendPort }}
  listeners:
  - name: staticreply
    address:
      socket_address:
        address: 127.0.0.1
        port_value: {{ .Vars.BackendPort }}
    filter_chains:
    - filters:
      - name: envoy.http_connection_manager
        config:
          stat_prefix: staticreply
          codec_type: auto
          route_config:
            name: staticreply
            virtual_hosts:
            - name: staticreply
              domains: ["*"]
              routes:
              - match:
                  prefix: "/"
                direct_response:
                  status: 200
                  body:
                    inline_string: "hello, world!"
          http_filters:
          - name: envoy.router
            config: {}
`

const ClientHTTPListener = `
name: client
traffic_direction: OUTBOUND
address:
  socket_address:
    address: 127.0.0.1
    port_value: {{ .Vars.ClientPort }}
filter_chains:
- filters:
  - name: envoy.http_connection_manager
    config:
      codec_type: AUTO
      stat_prefix: client
      http_filters:
      - name: envoy.router
      route_config:
        name: client
        virtual_hosts:
        - name: client
          domains: ["*"]
          routes:
          - match: { prefix: / }
            route:
              cluster: server
              timeout: 0s
`

const ServerHTTPListener = `
name: server
traffic_direction: INBOUND
address:
  socket_address:
    address: 127.0.0.1
    port_value: {{ .Vars.ServerPort }}
filter_chains:
- filters:
  - name: envoy.http_connection_manager
    config:
      codec_type: AUTO
      stat_prefix: server
      http_filters:
      - name: envoy.router
      route_config:
        name: server
        virtual_hosts:
        - name: server
          domains: ["*"]
          routes:
          - match: { prefix: / }
            route:
              cluster: inbound|9080|http|server.default.svc.cluster.local
              timeout: 0s
`

func TestBasicHTTP(t *testing.T) {
	ports := env.NewPorts(env.BasicHTTP)
	if err := (&Scenario{
		[]Step{
			&XDS{},
			&Update{Node: "client", Version: "0", Listeners: []string{ClientHTTPListener}},
			&Update{Node: "server", Version: "0", Listeners: []string{ServerHTTPListener}},
			&Envoy{Bootstrap: ServerBootstrap},
			&Envoy{Bootstrap: ClientBootstrap},
			&Sleep{1 * time.Second},
			&Get{ports.ClientToServerProxyPort, "hello, world!"},
		},
	}).Run(&Params{
		Vars: map[string]string{
			"BackendPort": fmt.Sprintf("%d", ports.BackendPort),
			"ClientPort":  fmt.Sprintf("%d", ports.ClientToServerProxyPort),
			"ClientAdmin": fmt.Sprintf("%d", ports.ClientAdminPort),
			"ServerAdmin": fmt.Sprintf("%d", ports.ServerAdminPort),
			"ServerPort":  fmt.Sprintf("%d", ports.ProxyToServerProxyPort),
		},
		XDS: int(ports.XDSPort),
	}); err != nil {
		t.Fatal(err)
	}
}