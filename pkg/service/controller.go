package service

import (
	"strconv"

	"github.com/container-storage-interface/spec/lib/go/csi"
	ovirtsdk "github.com/ovirt/go-ovirt"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ParameterStorageDomainName = "storageDomainName"
	ParameterThinProvisioning  = "thinProvisioning"
)

//ControllerService implements the controller interface
type ControllerService struct {
	ovirtClient *OvirtClient
	client      client.Client
}

var ControllerCaps = []csi.ControllerServiceCapability_RPC_Type{
	csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
	csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME, // attach/detach
}

//CreateVolume creates the disk for the request, unattached from any VM
func (c *ControllerService) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	klog.Infof("Creating disk %s", req.Name)
	// idempotence first - see if disk already exists, ovirt creates disk by name(alias in ovirt as well)
	conn, err := c.ovirtClient.GetConnection()
	if err != nil {
		klog.Errorf("Failed to get ovirt client connection")
		return nil, err
	}

	diskByName, err := conn.SystemService().DisksService().List().Search(req.Name).Send()
	if err != nil {
		return nil, err
	}

	// if exists we're done
	if disks, ok := diskByName.Disks(); ok && len(disks.Slice()) == 1 {
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

	// TODO rgolan the default in case of error would be non thin - change it?
	thinProvisioning, _ := strconv.ParseBool(req.Parameters[ParameterThinProvisioning])

	// creating the disk
	disk, err := ovirtsdk.NewDiskBuilder().
		Name(req.Name).
		StorageDomainsBuilderOfAny(*ovirtsdk.NewStorageDomainBuilder().Name(req.Parameters[ParameterStorageDomainName])).
		ProvisionedSize(req.CapacityRange.GetRequiredBytes()).
		ReadOnly(false).
		Format(ovirtsdk.DISKFORMAT_COW).
		Sparse(thinProvisioning).
		Build()

	if err != nil {
		// failed to construct the disk
		return nil, err
	}

	createDisk, err := conn.SystemService().DisksService().
		Add().
		Disk(disk).
		Send()
	if err != nil {
		// failed to create the disk
		klog.Errorf("Failed creating disk %s", req.Name)
		return nil, err
	}
	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			CapacityBytes: createDisk.MustDisk().MustProvisionedSize(),
			VolumeId:      createDisk.MustDisk().MustId(),
		},
	}, nil
}

//DeleteVolume removed the disk from oVirt
func (c *ControllerService) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	klog.Infof("Removing disk %s", req.VolumeId)
	// idempotence first - see if disk already exists, ovirt creates disk by name(alias in ovirt as well)
	conn, err := c.ovirtClient.GetConnection()
	if err != nil {
		klog.Errorf("Failed to get ovirt client connection")
		return nil, err
	}

	diskService := conn.SystemService().DisksService().DiskService(req.VolumeId)

	_, err = diskService.Get().Send()
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

	klog.Infof("Attaching Disk %s to VM %s", req.VolumeId, req.NodeId)
	conn, err := c.ovirtClient.GetConnection()
	if err != nil {
		klog.Errorf("Failed to get ovirt client connection")
		return nil, err
	}

	vmService := conn.SystemService().VmsService().VmService(req.NodeId)

	attachmentBuilder := ovirtsdk.NewDiskAttachmentBuilder().
		DiskBuilder(ovirtsdk.NewDiskBuilder().Id(req.VolumeId)).
		Interface(ovirtsdk.DISKINTERFACE_VIRTIO_SCSI).
		Bootable(false).
		Active(true)

	_, err = vmService.
		DiskAttachmentsService().
		Add().
		Attachment(attachmentBuilder.MustBuild()).
		Send()
	if err != nil {
		return nil, err
	}
	klog.Infof("Attached Disk %v to VM %s", req.VolumeId, req.NodeId)
	return &csi.ControllerPublishVolumeResponse{}, nil
}

//ControllerUnpublishVolume detaches the disk from the VM.
func (c *ControllerService) ControllerUnpublishVolume(_ context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	klog.Infof("Detaching Disk %s from VM %s", req.VolumeId, req.NodeId)
	conn, err := c.ovirtClient.GetConnection()
	if err != nil {
		klog.Errorf("Failed to get ovirt client connection ", err)
		return nil, err
	}

	attachment, err := diskAttachmentByVmAndDisk(conn, req.NodeId, req.VolumeId)
	if err != nil {
		klog.Errorf("Failed to get disk attachment %s for VM %s", req.VolumeId, req.NodeId)
		return nil, err
	}
	_, err = conn.SystemService().VmsService().VmService(req.NodeId).
		DiskAttachmentsService().
		AttachmentService(attachment.MustId()).
		Remove().
		Send()

	if err != nil {
		return nil, err
	}
	return &csi.ControllerUnpublishVolumeResponse{}, nil
}

//ValidateVolumeCapabilities
func (c *ControllerService) ValidateVolumeCapabilities(context.Context, *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

//ListVolumes
func (c *ControllerService) ListVolumes(context.Context, *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

//GetCapacity
func (c *ControllerService) GetCapacity(context.Context, *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

//CreateSnapshot
func (c *ControllerService) CreateSnapshot(context.Context, *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

//DeleteSnapshot
func (c *ControllerService) DeleteSnapshot(context.Context, *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

//ListSnapshots
func (c *ControllerService) ListSnapshots(context.Context, *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

//ControllerExpandVolume
func (c *ControllerService) ControllerExpandVolume(context.Context, *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

//ControllerGetCapabilities
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
