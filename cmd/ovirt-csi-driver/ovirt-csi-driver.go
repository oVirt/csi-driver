package main

import (
	"flag"
	"math/rand"
	"os"
	"time"

	ovirt "github.com/ovirt/csi-driver/internal"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/ovirt/csi-driver/pkg/service"
)

var (
	endpoint            = flag.String("endpoint", "unix:/csi/csi.sock", "CSI endpoint")
	namespace           = flag.String("namespace", "", "Namespace to run the controllers on")
	ovirtConfigFilePath = flag.String("ovirt-conf", "", "Path to ovirt api config")
	nodeName            = flag.String("node-name", "", "The node name - the node this pods runs on")
)

func init() {
	flag.Set("logtostderr", "true")
	klog.InitFlags(flag.CommandLine)
}

func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	handle()
	os.Exit(0)
}

func handle() {
	if service.VendorVersion == "" {
		klog.Fatalf("VendorVersion must be set at compile time")
	}
	klog.V(2).Infof("Driver vendor %v %v", service.VendorName, service.VendorVersion)

	ovirtClient, err := ovirt.NewOvirtClient()
	if err != nil {
		klog.Fatalf("Failed to initialize ovirt client %s", err)
	}

	// Get a config to talk to the apiserver
	restConfig, err := config.GetConfig()
	if err != nil {
		klog.Fatal(err)
	}

	opts := manager.Options{
		Namespace: *namespace,
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(restConfig, opts)
	if err != nil {
		klog.Fatal(err)
	}

	// get the node object by name and pass the VM ID because it is the node
	// id from the storage perspective. It will be used for attaching disks
	var nodeId string
	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		klog.Fatal(err)
	}

	if *nodeName != "" {
		get, err := clientSet.CoreV1().Nodes().Get(*nodeName, metav1.GetOptions{})
		if err != nil {
			klog.Fatal(err)
		}
		nodeId = get.Status.NodeInfo.SystemUUID
	}

	driver := service.NewOvirtCSIDriver(ovirtClient, mgr.GetClient(), nodeId)

	driver.Run(*endpoint)
}
