package service

import (
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// set by ldflags
	VendorVersion = "0.1.1"
	VendorName    = "csi.ovirt.org"
)

type OvirtCSIDriver struct {
	*IdentityService
	*ControllerService
	*NodeService
	nodeId string
	ovirtClient *OvirtClient
	Client      client.Client
}

// NewOvirtCSIDriver creates a driver instance
func NewOvirtCSIDriver(ovirtClient *OvirtClient, client client.Client, nodeId string) *OvirtCSIDriver {
	d := OvirtCSIDriver{
		IdentityService:   &IdentityService{},
		ControllerService: &ControllerService{ovirtClient: ovirtClient, client: client},
		NodeService:       &NodeService{nodeId: nodeId, ovirtClient: ovirtClient},
		ovirtClient:       ovirtClient,
		Client:            client,
	}
	return &d
}

// Run will initiate the grpc services Identity, Controller, and Node.
func (driver *OvirtCSIDriver) Run(endpoint string) {
	// run the gRPC server
	klog.Info("Setting the rpc server")

	s := NewNonBlockingGRPCServer()
	s.Start(endpoint, driver.IdentityService, driver.ControllerService, driver.NodeService)
	s.Wait()
}
