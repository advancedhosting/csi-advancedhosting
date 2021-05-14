/*
Copyright 2021 Advanced Hosting

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package driver

import (
	"context"
	"fmt"
	"github.com/advancedhosting/advancedhosting-api-go/ah"
	"net"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc"
	"k8s.io/klog"
)

const (
	DriverName                = "csi.advancedhosting.com"
	DriverVersion             = "1.0.0"
	TopologySegmentDatacenter = DriverName + "/datacenter"
)

type DriverOptions struct {
	Endpoint string
	Url      string
	Token    string
}

// Driver implements the csi driver according the spec
type Driver struct {
	endpoint string
	*controllerService
	*nodeService
}

func NewDriver(options *DriverOptions) (*Driver, error) {

	clientOptions := &ah.ClientOptions{
		Token:   options.Token,
		BaseURL: options.Url,
	}

	client, err := ah.NewAPIClient(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("an error occurred while creating api client")
	}

	driverControllerService := NewControllerService(client)
	driverNodeService, err := NewNodeService(client)
	if err != nil {
		return nil, fmt.Errorf("error creating driverNodeService: %v", err)
	}

	driver := &Driver{
		endpoint:          options.Endpoint,
		controllerService: driverControllerService,
		nodeService:       driverNodeService,
	}

	return driver, nil
}

func (d *Driver) Run() error {

	endpoint, err := url.Parse(d.endpoint)
	if err != nil {
		return err
	}

	if endpoint.Scheme != "unix" {
		return fmt.Errorf("Scheme %s is not supported", endpoint.Scheme)
	}

	grpcAddress := path.Join(endpoint.Host, filepath.FromSlash(endpoint.Path))

	// Remove existing socket
	if err := os.Remove(grpcAddress); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Error removing socket %s", grpcAddress)
	}

	listener, err := net.Listen(endpoint.Scheme, grpcAddress)
	if err != nil {
		return err
	}

	logErr := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			klog.Errorf("GRPC error in %s: %v", info.FullMethod, err)
		}
		return resp, err
	}
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(logErr),
	}
	grpcServer := grpc.NewServer(opts...)

	csi.RegisterIdentityServer(grpcServer, d)
	csi.RegisterControllerServer(grpcServer, d)
	csi.RegisterNodeServer(grpcServer, d)

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stopCh
		grpcServer.GracefulStop()
	}()

	klog.Info("Starting GRPC")
	return grpcServer.Serve(listener)

}
