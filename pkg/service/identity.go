package service

import (
	"github.com/container-storage-interface/spec/lib/go/csi"
	"golang.org/x/net/context"
	"k8s.io/klog"
)

//IdentityService of ovirt-csi-driver
type IdentityService struct{}

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

//Probe
func (i *IdentityService) Probe(ctx context.Context, request *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	klog.V(4).Infof("probe called with args: %#v", request)
	return &csi.ProbeResponse{}, nil
}
