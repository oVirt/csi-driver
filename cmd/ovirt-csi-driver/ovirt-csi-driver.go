package main

import (
	"flag"
	"math/rand"
	"os"
	"time"

	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/ovirt/csi-driver/pkg/service"
)

var (
	endpoint            = flag.String("endpoint", "unix:/csi/csi.sock", "CSI endpoint")
	namespace           = flag.String("namespace", "", "Namespace to run the controllers on")
	ovirtConfigFilePath = flag.String("ovirt-conf", "", "Path to ovirt api config")
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

	ovirtClient, err := service.NewOvirtClient()
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

	driver := service.NewOvirtCSIDriver(ovirtClient, mgr.GetClient())

	driver.Run(*endpoint)
}


