package service

import (
	"strconv"

	"github.com/container-storage-interface/spec/lib/go/csi"
	ovirtsdk "github.com/ovirt/go-ovirt"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	apicorev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ParameterStorageDomainName    = "ovirtStorageDomain"
	ParameterDiskThinProvisioning = "ovirtDiskThinProvisioning"
	ParameterFsType               = "fsType"
)

type ControllerService struct {
	ovirtClient *OvirtClient
	client      client.Client
}

var ControllerCaps = []csi.ControllerServiceCapability_RPC_Type{
	csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
	csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME, // attach/detach
}

func (c *ControllerService) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	klog.Infof("Creating disk %s", req.Name)
	// idempotence first - see if disk already exists, ovirt creates disk by name(alias in ovirt as well)
	diskByName, err := c.ovirtClient.connection.SystemService().DisksService().List().Search(req.Name).Send()
	if err != nil {
		return nil, err
	}

	// if exists we're done
	if disks, ok := diskByName.Disks(); ok {
		disk := disks.Slice()[0]
		return &csi.CreateVolumeResponse{
			Volume: &csi.Volume{
				CapacityBytes:      disk.MustProvisionedSize(),
				VolumeId:           disk.MustId(),
				VolumeContext:      nil,
				ContentSource:      nil,
				AccessibleTopology: nil,
			},
		}, nil
	}

	// TODO rgolan the default incase of error would be non thin - change it?
	thinProvisioning, _ := strconv.ParseBool(req.Parameters[ParameterDiskThinProvisioning])

	// creating the disk
	disk, err := ovirtsdk.NewDiskBuilder().
		Name(req.Name).
		StorageDomainsBuilderOfAny(*ovirtsdk.NewStorageDomainBuilder().Name(req.Parameters[ParameterStorageDomainName])).
		ProvisionedSize(req.CapacityRange.GetRequiredBytes()).
		ReadOnly(false).
		Format(ovirtsdk.DISKFORMAT_COW).
		Sparse(thinProvisioning).Build()

	if err != nil {
		// failed to construct the disk
		return nil, err
	}

	klog.Infof("Finished creating disk with ID %s", disk.MustId())
	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			CapacityBytes: disk.MustProvisionedSize(),
			VolumeId:      disk.MustId(),
		},
	}, nil
}

func (c *ControllerService) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	klog.Infof("Removing disk %s", req.VolumeId)
	// idempotence first - see if disk already exists, ovirt creates disk by name(alias in ovirt as well)
	diskService := c.ovirtClient.connection.SystemService().DisksService().DiskService(req.VolumeId)

	_, err := diskService.Get().Send()
	// if doesn't exists we're done
	if err != nil {
		return &csi.DeleteVolumeResponse{}, nil
	}
	_, err = diskService.Remove().Send()
	if err != nil {
		return nil, err
	}

	klog.Infof("Finished removing disk %s", req.VolumeId)
	return &csi.DeleteVolumeResponse{}, nil
}

// ControllerPublishVolume takes a volume, which is an oVirt disk, and attaches it to a node, which is an oVirt VM.
func (c *ControllerService) ControllerPublishVolume(
	ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	// get the vm ID by the node name
	// 1. get the kuberetes api client

	// 2. fetch the node object by node it
	var node apicorev1.Node
	key := client.ObjectKey{Namespace: "", Name: req.NodeId}
	err := c.client.Get(ctx, key, &node)
	if err != nil {
		return nil, err
	}

	// 3. use machineId or systemUUID
	vmId := node.Status.NodeInfo.MachineID
	vmService := c.ovirtClient.connection.SystemService().VmsService().VmService(vmId)
	attachmentBuilder := ovirtsdk.NewDiskAttachmentBuilder().Id(req.VolumeId)
	_, err = vmService.DiskAttachmentsService().AddProvidingDiskId().Attachment(attachmentBuilder.MustBuild()).Send()
	if err != nil {
		return nil, err
	}
	klog.Infof("Attached Disk ID %v to VM ID %s", req.VolumeId, vmId)
	return &csi.ControllerPublishVolumeResponse{}, nil
}

func (c *ControllerService) ControllerUnpublishVolume(context.Context, *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (c *ControllerService) ValidateVolumeCapabilities(context.Context, *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (c *ControllerService) ListVolumes(context.Context, *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (c *ControllerService) GetCapacity(context.Context, *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (c *ControllerService) CreateSnapshot(context.Context, *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (c *ControllerService) DeleteSnapshot(context.Context, *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (c *ControllerService) ListSnapshots(context.Context, *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (c *ControllerService) ControllerExpandVolume(context.Context, *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (c *ControllerService) ControllerGetCapabilities(context.Context, *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	caps := make([]*csi.ControllerServiceCapability, 0, len(ControllerCaps))
	for _, capability := range ControllerCaps {
		caps = append(
			caps,
			&csi.ControllerServiceCapability{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{
						Type: capability,
					},
				},
			},
		)
	}
	return &csi.ControllerGetCapabilitiesResponse{Capabilities: caps}, nil
}
