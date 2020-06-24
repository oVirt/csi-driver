package service

import (
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/ovirt/csi-driver/internal/ovirt"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"
)

//IdentityService of ovirt-csi-driver
type IdentityService struct {
	ovirtClient ovirt.Client
}

//GetPluginInfo returns the vendor name and version - set in build time
func (i *IdentityService) GetPluginInfo(context.Context, *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	return &csi.GetPluginInfoResponse{
		Name:          VendorName,
		VendorVersion: VendorVersion,
	}, nil
}

//GetPluginCapabilities declares the plugins capabilities
func (i *IdentityService) GetPluginCapabilities(context.Context, *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	return &csi.GetPluginCapabilitiesResponse{
		Capabilities: []*csi.PluginCapability{
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_CONTROLLER_SERVICE,
					},
				},
			},
		},
	}, nil
}

// Probe checks the state of the connection to ovirt-engine
func (i *IdentityService) Probe(ctx context.Context, request *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	c, err := i.ovirtClient.GetConnection()
	if err != nil {
		klog.Errorf("Could not get connection %v", err)
		return nil, status.Error(codes.FailedPrecondition, "Could not get connection to ovirt-engine")
	}

	if err := c.Test(); err != nil {
		klog.Errorf("Connection test failed %v", err)
		return nil, status.Error(codes.FailedPrecondition, "Could not get connection to ovirt-engine")
	}

	return &csi.ProbeResponse{Ready: &wrappers.BoolValue{Value: true}}, nil
}
