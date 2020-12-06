package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/utils/mount"

	"github.com/container-storage-interface/spec/lib/go/csi"
	ovirtsdk "github.com/ovirt/go-ovirt"
	"golang.org/x/net/context"
	"k8s.io/klog"

	"github.com/ovirt/csi-driver/internal/ovirt"
)

var NodeCaps = []csi.NodeServiceCapability_RPC_Type{
	csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
}

type NodeService struct {
	nodeId      string
	ovirtClient *ovirt.Client
	deviceLister       deviceLister
	fsMaker            fsMaker
	fsMounter          mount.Interface
	dirMaker           dirMaker
}

type deviceLister interface{ List() ([]byte, error) }
type fsMaker interface { Make(device string, fsType string) error }
type dirMaker interface { Make(path string, perm os.FileMode) error }

func NewNodeService(nodeId string, ovirtClient *ovirt.Client) *NodeService {
	return &NodeService{
		nodeId: nodeId,
		ovirtClient: ovirtClient,
		deviceLister: deviceListerFunc(func() ([]byte, error) {
			return exec.Command("lsblk", "-nJo", "SERIAL,PATH,FSTYPE").Output()
		}),
		fsMaker: fsMakerFunc(func(device, fsType string) error {
			return makeFS(device, fsType)
		}),
		fsMounter: mount.New(""),
		dirMaker: dirMakerFunc(func(path string, perm os.FileMode) error {
			// MkdirAll returns nil if path already exists
			return os.MkdirAll(path, perm)
		}),
	}
}

func (n *NodeService) NodeStageVolume(_ context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	klog.Infof("Staging volume %s with %+v", req.VolumeId, req)
	if req.VolumeCapability.GetBlock() != nil {
		klog.Infof("Volume %s is a block volume, no need for staging", req.VolumeId)
		return &csi.NodeStageVolumeResponse{}, nil
	}

	conn, err := n.ovirtClient.GetConnection()
	if err != nil {
		klog.Errorf("Failed to get ovirt client connection")
		return nil, err
	}

	device, err := getDeviceByAttachmentId(req.VolumeId, n.nodeId, conn, n.deviceLister)
	if err != nil {
		klog.Errorf("Failed to fetch device by attachment-id for volume %s on node %s", req.VolumeId, n.nodeId)
		return nil, err
	}

	// is there a filesystem on this device?
	if device.FSType != "" {
		klog.Infof("Detected fs %s, returning", device.FSType)
		return &csi.NodeStageVolumeResponse{}, nil
	}

	fsType := req.VolumeCapability.GetMount().FsType
	// no filesystem - create it
	klog.Infof("Creating FS %s on device %s", fsType, device)
	err = n.fsMaker.Make(device.Path, fsType)
	if err != nil {
		klog.Errorf("Could not create filesystem %s on %s", fsType, device.Path)
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

	device, err := getDeviceByAttachmentId(req.VolumeId, n.nodeId, conn, n.deviceLister)
	if err != nil {
		klog.Errorf("Failed to fetch device by attachment-id for volume %s on node %s", req.VolumeId, n.nodeId)
		return nil, err
	}

	if req.VolumeCapability.GetBlock() != nil {
		return n.publishBlockVolume(req, device.Path)
	}

	targetPath := req.GetTargetPath()
	err = n.dirMaker.Make(targetPath, 0644)
	if err != nil {
		return nil, errors.New(err.Error())
	}

	fsType := req.VolumeCapability.GetMount().FsType
	klog.Infof("Mounting devicePath %s, on targetPath: %s with FS type: %s",
		device.Path, targetPath, fsType)
	err = n.fsMounter.Mount(device.Path, targetPath, fsType, []string{})
	if err != nil {
		klog.Errorf("Failed mounting %v", err)
		return nil, err
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (n *NodeService) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	klog.Infof("Unmounting %s", req.GetTargetPath())
	err := n.fsMounter.Unmount(req.GetTargetPath())
	if err != nil {
		klog.Infof("Failed to unmount")
		return nil, err
	}

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (n *NodeService) publishBlockVolume(req *csi.NodePublishVolumeRequest, device string) (*csi.NodePublishVolumeResponse, error) {
	klog.Infof("Publishing block volume, device: %s, req: %+v", device, req)
	file, err := os.OpenFile(req.TargetPath, os.O_CREATE, os.FileMode(0644))
	defer file.Close()
	if err != nil {
		if !os.IsExist(err) {
			return nil, status.Errorf(codes.Internal, "Failed to create targetPath %s, err: %v", req.TargetPath, err)
		}
	}

	mounter := mount.New("")
	err = mounter.Mount(device, req.TargetPath, "", []string{"bind"})
	if err != nil {
		if removeErr := os.Remove(req.TargetPath); removeErr != nil {
			return nil, status.Errorf(codes.Internal, "Failed to remove mount target %v, err: %v, mount error: %v", req.TargetPath, removeErr, err)
		}

		return nil, status.Errorf(codes.Internal, "Failed to mount %v at %v, err: %v", device, req.TargetPath, err)
	}

	return &csi.NodePublishVolumeResponse{}, nil
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

func getDeviceByAttachmentId(volumeID, nodeID string, conn *ovirtsdk.Connection, deviceLister deviceLister) (device, error) {
	attachment, err := diskAttachmentByVmAndDisk(conn, nodeID, volumeID)
	if err != nil {
		return device{}, err
	}

	klog.Infof("Extracting pvc volume name %s", volumeID)
	disk, _ := conn.FollowLink(attachment.MustDisk())
	d, ok := disk.(*ovirtsdk.Disk)
	if !ok {
		return device{}, errors.New("couldn't retrieve disk from attachment")
	}
	klog.Infof("Extracted disk ID from PVC %s", d.MustId())

	// ovirt Disk ID = serial ID of the disk on the OS
	serialID := d.MustId()

	deviceByID, err := getDeviceBySerialID(serialID, deviceLister)
	if err != nil {
		klog.Errorf("Device with serial ID %s does not exists", serialID)
		return device{}, errors.New("device was not found")
	}
	return deviceByID, nil
}

type devices struct {
	BlockDevices []device `json:"blockdevices"`
}

type device struct {
	SerialID string `json:"serial"`
	Path     string `json:"path"`
	FSType   string `json:"fstype"`
}

// getDeviceBySerialID reads the block devices details, serialID, path, and FS type, and
// returns the device that matches the serialID.
func getDeviceBySerialID(serialID string, deviceLister deviceLister) (device, error) {
	klog.Infof("Get the device details by serialID %s", serialID)
	klog.Info("lsblk -nJo SERIAL,PATH,FSTYPE")

	out, err := deviceLister.List()
	exitError, incompleteCmd := err.(*exec.ExitError)
	if err != nil && incompleteCmd {
		return device{}, errors.New(err.Error() + "lsblk failed with " + string(exitError.Stderr))
	}

	devices := devices{}
	err = json.Unmarshal(out, &devices)
	if err != nil {
		klog.Errorf("Failed to parse json output from lsblk: %s", err)
		return device{}, err
	}

	for _, d := range devices.BlockDevices {
		if d.SerialID == serialID {
			return d, nil
		}
	}
	return device{}, errors.New("couldn't find device by serial id")
}

func makeFS(devicePath string, fsType string) error {
	// caution, use force flag when creating the filesystem if it doesn't exit.
	klog.Infof("Mounting devicePath %s, with FS %s", devicePath, fsType)

	var cmd *exec.Cmd
	var stdout, stderr bytes.Buffer
	if strings.HasPrefix(fsType, "ext") {
		cmd = exec.Command("mkfs", "-F", "-t", fsType, devicePath)
	} else if strings.HasPrefix(fsType, "xfs") {
		cmd = exec.Command("mkfs", "-t", fsType, "-f", devicePath)
	} else {
		return errors.New(fsType + " is not supported, only xfs and ext are supported")
	}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitError, incompleteCmd := err.(*exec.ExitError)
	if err != nil && incompleteCmd {
		klog.Errorf("stdout: %s", string(stdout.Bytes()))
		klog.Errorf("stderr: %s", string(stderr.Bytes()))
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

type deviceListerFunc func() ([]byte, error)
func (d deviceListerFunc) List() ([]byte, error) {
	return d()
}
type fsMakerFunc func(device, fsType string) error
func (f fsMakerFunc) Make(device, fsType string) error {
	return f(device, fsType)
}

type dirMakerFunc func(path string, perm os.FileMode) error
func (d dirMakerFunc) Make(path string, perm os.FileMode) error {
	return d(path, perm)
}
