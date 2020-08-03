// Copyright Istio Authors
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

package extension_server

import (
	"context"

	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	extensionservice "github.com/envoyproxy/go-control-plane/envoy/service/extension/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

const (
	// ApiType for extension configs.
	ApiType = "type.googleapis.com/envoy.config.core.v3.TypedExtensionConfig"
)

// ExtensionServer is the main server instance.
type ExtensionServer struct {
	server.Server
	server.CallbackFuncs
	cache *cache.LinearCache
}

var _ extensionservice.ExtensionConfigDiscoveryServiceServer = &ExtensionServer{}

func New(ctx context.Context) *ExtensionServer {
	out := &ExtensionServer{}
	out.cache = cache.NewLinearCache(ApiType, nil)
	out.Server = server.NewServer(ctx, out.cache, out)
	return out
}

func (es *ExtensionServer) StreamExtensionConfigs(stream extensionservice.ExtensionConfigDiscoveryService_StreamExtensionConfigsServer) error {
	return status.Errorf(codes.Unimplemented, "not implemented")
}
func (es *ExtensionServer) DeltaExtensionConfigs(_ extensionservice.ExtensionConfigDiscoveryService_DeltaExtensionConfigsServer) error {
	return status.Errorf(codes.Unimplemented, "not implemented")
}
func (es *ExtensionServer) FetchExtensionConfigs(ctx context.Context, req *discovery.DiscoveryRequest) (*discovery.DiscoveryResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.Unavailable, "empty request")
	}
	req.TypeUrl = ApiType
	return es.Server.Fetch(ctx, req)
}