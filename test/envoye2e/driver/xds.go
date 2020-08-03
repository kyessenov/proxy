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
	"context"
	"fmt"
	"log"
	"net"

	"istio.io/proxy/tools/extension_server"

	cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	extensionservice "github.com/envoyproxy/go-control-plane/envoy/service/extension/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"google.golang.org/grpc"
)

// XDS creates an xDS server
type XDS struct {
	grpc            *grpc.Server
	extensionserver *extension_server.ExtensionServer
}

var _ Step = &XDS{}

func (x *XDS) Run(p *Params) error {
	log.Printf("XDS server starting on %d\n", p.XDS)
	x.grpc = grpc.NewServer()
	x.extensionserver = extension_server.New(context.Background())
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", p.XDS))
	if err != nil {
		return err
	}

	p.Config = cache.NewSnapshotCache(false, cache.IDHash{}, x)
	xdsServer := server.NewServer(context.Background(), p.Config, nil)
	discovery.RegisterAggregatedDiscoveryServiceServer(x.grpc, xdsServer)
	extensionservice.RegisterExtensionConfigDiscoveryServiceServer(x.grpc, x.extensionserver)

	go func() {
		_ = x.grpc.Serve(lis)
	}()
	return nil
}

func (x *XDS) Cleanup() {
	log.Println("stopping XDS server")
	x.grpc.GracefulStop()
}
func (x *XDS) Debugf(format string, args ...interface{}) {
	log.Printf("xds debug: "+format, args...)
}
func (x *XDS) Infof(format string, args ...interface{}) {
	log.Printf("xds: "+format, args...)
}
func (x *XDS) Errorf(format string, args ...interface{}) {
	log.Printf("xds error: "+format, args...)
}
func (x *XDS) Warnf(format string, args ...interface{}) {
	log.Printf("xds warn: "+format, args...)
}

type Update struct {
	Node      string
	Version   string
	Listeners []string
	Clusters  []string
}

var _ Step = &Update{}

func (u *Update) Run(p *Params) error {
	p.Vars["Version"] = u.Version
	version, err := p.Fill(u.Version)
	if err != nil {
		return err
	}
	log.Printf("update config for %q with version %q", u.Node, version)

	clusters := make([]types.Resource, 0, len(u.Clusters))
	for _, cluster := range u.Clusters {
		out := &cluster_v3.Cluster{}
		if err := p.FillYAML(cluster, out); err != nil {
			return err
		}
		clusters = append(clusters, out)
	}

	listeners := make([]types.Resource, 0, len(u.Listeners))
	for _, listener := range u.Listeners {
		out := &listener_v3.Listener{}
		if err := p.FillYAML(listener, out); err != nil {
			return err
		}
		listeners = append(listeners, out)
	}

	snap := cache.Snapshot{}
	snap.Resources[types.Cluster] = cache.NewResources(version, clusters)
	snap.Resources[types.Listener] = cache.NewResources(version, listeners)
	return p.Config.SetSnapshot(u.Node, snap)
}

func (u *Update) Cleanup() {}
