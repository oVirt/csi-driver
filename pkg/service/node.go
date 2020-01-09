package service

import (
	"bufio"
	"bytes"
	"errors"
	"os/exec"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
	ovirtsdk "github.com/ovirt/go-ovirt"
	"golang.org/x/net/context"
	"golang.org/x/sys/unix"
	"k8s.io/klog"
)

type NodeService struct {
	nodeId string
	ovirtClient *OvirtClient
}

var NodeCaps = []csi.NodeServiceCapability_RPC_Type{
	csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
}

func devFromVolumeId(id string, diskInterface ovirtsdk.DiskInterface) (string, error) {
	switch diskInterface {
	case ovirtsdk.DISKINTERFACE_VIRTIO:
		return "/dev/disk/by-id/virtio-" + id, nil
	case ovirtsdk.DISKINTERFACE_VIRTIO_SCSI:
		return "/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_" + id, nil
	}
	return "", errors.New("device type is unsupported")
}

func (n *NodeService) NodeStageVolume(_ context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	// mount
	// Get the real disk device name from device
	klog.Infof("Staging volume %s with %+v", req.VolumeId, req)
	attachment, err := diskAttachmentByVmAndDisk(n.ovirtClient.connection, n.nodeId, req.VolumeId)
	if err != nil {
		return nil, err
	}

	device, err := devFromVolumeId(req.VolumeId, attachment.MustInterface())
	if err != nil {
		return nil, err
	}

	// is there a filesystem on this device?
	filesystem, err := getDeviceInfo(device)
	if err != nil {
		return nil, err
	}
	//TODO get the desired fstype from req.PublishRequest[]
	if filesystem == "" {
		// no filesystem - create it
		makeFSErr := makeFS(device, "ext4")
		if makeFSErr != nil {
			return nil, makeFSErr
		}
	}
	return &csi.NodeStageVolumeResponse{}, nil
}

func (n *NodeService) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	if !isMountpoint(req.StagingTargetPath) {
		// nothing to do, return.
		return &csi.NodeUnstageVolumeResponse{}, nil
	}

	err := unix.Unmount(req.StagingTargetPath, 0)
	if err != nil {
		return nil, err
	}
	return &csi.NodeUnstageVolumeResponse{}, nil

}

func (n *NodeService) NodePublishVolume(context.Context, *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	panic("implement me")
}

func (n *NodeService) NodeUnpublishVolume(context.Context, *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	panic("implement me")
}

func (n *NodeService) NodeGetVolumeStats(context.Context, *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	panic("implement me")
}

func (n *NodeService) NodeExpandVolume(context.Context, *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	panic("implement me")
}

func (n *NodeService) NodeGetInfo(context.Context, *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	return &csi.NodeGetInfoResponse{NodeId: n.nodeId}, nil
}

func (n *NodeService) NodeGetCapabilities(context.Context, *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	caps := make([]*csi.NodeServiceCapability, 0, len(NodeCaps))
	for _, c := range NodeCaps {
		caps = append(
			caps,
			&csi.NodeServiceCapability{
				Type: &csi.NodeServiceCapability_Rpc{
					Rpc: &csi.NodeServiceCapability_RPC{
						Type: c,
					},
				},
			},
		)
	}
	return &csi.NodeGetCapabilitiesResponse{Capabilities: caps}, nil
}

// getDeviceInfo will return the first Device which is a partition and its filesystem.
// if the given Device disk has no partition then an empty zero valued device will return
func getDeviceInfo(device string) (string, error) {
	cmd := exec.Command("lsblk", "-nro", "FSTYPE", device)
	out, err := cmd.Output()
	exitError, incompleteCmd := err.(*exec.ExitError)
	if err != nil && incompleteCmd {
		return "", errors.New(err.Error() + "lsblk failed with " + string(exitError.Stderr))
	}

	reader := bufio.NewReader(bytes.NewReader(out))
	line, _, err := reader.ReadLine()
	if err != nil {
		return "", err
	}
	return string(line), nil
}

func makeFS(device string, fsType string) error {
	// caution, use -F to force creating the filesystem if it doesn't exit. May not be portable for fs other
	// than ext family
	var force string
	if strings.HasPrefix(fsType, "ext") {
		force = "-F"
	}
	cmd := exec.Command("mkfs", force, "-t", fsType, device)
	err := cmd.Run()
	exitError, incompleteCmd := err.(*exec.ExitError)
	if err != nil && incompleteCmd {
		return errors.New(err.Error() + " mkfs failed with " + string(exitError.Error()))
	}
	return nil
}

// isMountpoint find out if a given directory is a real mount point
func isMountpoint(mountDir string) bool {
	cmd := exec.Command("findmnt", mountDir)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return false
	}
	return true
}
