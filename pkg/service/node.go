package service

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"fmt"

	"k8s.io/utils/mount"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/ovirt/csi-driver/internal/ovirt"
	ovirtsdk "github.com/ovirt/go-ovirt"
	"golang.org/x/net/context"
	"k8s.io/klog"
)

type NodeService struct {
	nodeId      string
	ovirtClient *ovirt.Client
}

var NodeCaps = []csi.NodeServiceCapability_RPC_Type{
	csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
	csi.NodeServiceCapability_RPC_EXPAND_VOLUME,
}

func baseDevicePathByInterface(diskInterface ovirtsdk.DiskInterface) (string, error) {
	switch diskInterface {
	case ovirtsdk.DISKINTERFACE_VIRTIO:
		return "/dev/disk/by-id/virtio-", nil
	case ovirtsdk.DISKINTERFACE_VIRTIO_SCSI:
		return "/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_", nil
	}
	return "", errors.New("device type is unsupported")
}

func (n *NodeService) NodeStageVolume(_ context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	klog.Infof("Staging volume %s with %+v", req.VolumeId, req)
	conn, err := n.ovirtClient.GetConnection()
	if err != nil {
		klog.Errorf("Failed to get ovirt client connection")
		return nil, err
	}

	device, err := getDeviceByAttachmentId(req.VolumeId, n.nodeId, conn)
	if err != nil {
		klog.Errorf("Failed to fetch device by attachment-id for volume %s on node %s", req.VolumeId, n.nodeId)
		return nil, err
	}

	// is there a filesystem on this device?
	filesystem, err := getDeviceInfo(device)
	if err != nil {
		klog.Errorf("Failed to fetch device info for volume %s on node %s", req.VolumeId, n.nodeId)
		return nil, err
	}
	if filesystem != "" {
		klog.Infof("Detected fs %s, returning", filesystem)
		return &csi.NodeStageVolumeResponse{}, nil
	}

	fsType := req.VolumeCapability.GetMount().FsType
	// no filesystem - create it
	klog.Infof("Creating FS %s on device %s", fsType, device)
	err = makeFS(device, fsType)
	if err != nil {
		klog.Errorf("Could not create filesystem %s on %s", fsType, device)
		return nil, err
	}

	return &csi.NodeStageVolumeResponse{}, nil
}

func (n *NodeService) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (n *NodeService) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	conn, err := n.ovirtClient.GetConnection()
	if err != nil {
		klog.Errorf("Failed to get ovirt client connection")
		return nil, err
	}

	device, err := getDeviceByAttachmentId(req.VolumeId, n.nodeId, conn)
	if err != nil {
		klog.Errorf("Failed to fetch device by attachment-id for volume %s on node %s", req.VolumeId, n.nodeId)
		return nil, err
	}

	targetPath := req.GetTargetPath()
	err = os.MkdirAll(targetPath, 0750)
	if err != nil {
		return nil, errors.New(err.Error())
	}

	fsType := req.VolumeCapability.GetMount().FsType
	klog.Infof("Mounting devicePath %s, on targetPath: %s with FS type: %s",
		device, targetPath, fsType)
	mounter := mount.New("")
	err = mounter.Mount(device, targetPath, fsType, []string{})
	if err != nil {
		klog.Errorf("Failed mounting %v", err)
		return nil, err
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (n *NodeService) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	mounter := mount.New("")
	klog.Infof("Unmounting %s", req.GetTargetPath())
	err := mounter.Unmount(req.GetTargetPath())
	if err != nil {
		klog.Infof("Failed to unmount")
		return nil, err
	}

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (n *NodeService) NodeGetVolumeStats(context.Context, *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	panic("implement me")
}

func (n *NodeService) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	//fut
	klog.Infof("Resizing filesystem mounted on %s", req.VolumePath)
	fsType := req.VolumeCapability.GetMount().FsType
	klog.Infof("fstype: %s", fsType)

	var resizeCmd string
	if strings.HasPrefix(fsType, "ext") {
		resizeCmd = "resize2fs"
	}
	if strings.HasPrefix(fsType, "xfs") {
		resizeCmd = "xfs_growfs"
	}

	if resizeCmd == "" {
		return nil, fmt.Errorf("fsType is neither xfs or ext[234]")
	}

	device, err := getDeviceByMountPoint(req.VolumePath)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(resizeCmd, device)
	err = cmd.Run()
	exitError, incompleteCmd := err.(*exec.ExitError)
	if err != nil && incompleteCmd {
		return nil, errors.New(err.Error() + " resize failed with " + string(exitError.Error()))
	}

	klog.Infof("Resized %s filesystem on device %s)", fsType, device)
	return &csi.NodeExpandVolumeResponse{}, nil
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

func getDeviceByAttachmentId(volumeID, nodeID string, conn *ovirtsdk.Connection) (string, error) {
	attachment, err := diskAttachmentByVmAndDisk(conn, nodeID, volumeID)
	if err != nil {
		return "", err
	}

	klog.Infof("Extracting pvc volume name %s", volumeID)
	disk, _ := conn.FollowLink(attachment.MustDisk())
	d, ok := disk.(*ovirtsdk.Disk)
	if !ok {
		return "", errors.New("couldn't retrieve disk from attachment")
	}
	klog.Infof("Extracted disk ID from PVC %s", d.MustId())

	baseDevicePath, err := baseDevicePathByInterface(attachment.MustInterface())
	if err != nil {
		return "", err
	}

	// verify the device path exists
	device := baseDevicePath + d.MustId()
	_, err = os.Stat(device)
	if err == nil {
		klog.Infof("Device path %s exists", device)
		return device, nil
	}

	if os.IsNotExist(err) {
		// try with short disk ID, where the serial ID is only 20 chars long (controlled by udev)
		shortDevice := baseDevicePath + d.MustId()[:20]
		_, err = os.Stat(shortDevice)
		if err == nil {
			klog.Infof("Device path %s exists", shortDevice)
			return shortDevice, nil
		}
	}
	klog.Errorf("Device path for disk ID %s does not exists", d.MustId())
	return "", errors.New("device was not found")
}

// getDeviceInfo will return the first Device which is a partition and its filesystem.
// if the given Device disk has no partition then an empty zero valued device will return
func getDeviceInfo(device string) (string, error) {
	devicePath, err := filepath.EvalSymlinks(device)
	if err != nil {
		klog.Errorf("Unable to evaluate symlink for device %s", device)
		return "", errors.New(err.Error())
	}

	klog.Info("lsblk -nro FSTYPE ", devicePath)
	cmd := exec.Command("lsblk", "-nro", "FSTYPE", devicePath)
	out, err := cmd.Output()
	exitError, incompleteCmd := err.(*exec.ExitError)
	if err != nil && incompleteCmd {
		return "", errors.New(err.Error() + "lsblk failed with " + string(exitError.Stderr))
	}

	reader := bufio.NewReader(bytes.NewReader(out))
	line, _, err := reader.ReadLine()
	if err != nil {
		klog.Errorf("Error occured while trying to read lsblk output")
		return "", err
	}
	return string(line), nil
}

func makeFS(device string, fsType string) error {
	// caution, use -F to force creating the filesystem if it doesn't exit. May not be portable for fs other
	// than ext family
	klog.Infof("Mounting device %s, with FS %s", device, fsType)
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

func getDeviceByMountPoint(mp string) (string, error) {
    out, err := exec.Command("findmnt", "-nfc", mp).Output()
    if err != nil {
        return "", fmt.Errorf("Error: %v\n", err)
    }

    s := strings.Fields(string(out))
    if len(s) < 2 {
        return "", fmt.Errorf("Could not parse command output: >%s<", string(out))
    }
    return s[1], nil
}
