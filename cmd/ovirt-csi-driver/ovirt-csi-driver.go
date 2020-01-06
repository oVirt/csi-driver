package main

import (
	"errors"
	"flag"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	ovirtsdk "github.com/ovirt/go-ovirt"
	"gopkg.in/yaml.v2"
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

	ovirtConnection, err := newOvirtConnection()
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

	driver := service.NewOvirtCSIDriver(ovirtConnection, mgr.GetClient())

	driver.Run(*endpoint)
}

func newOvirtConnection() (*ovirtsdk.Connection, error) {

	ovirtConfig, err := GetOvirtConfig()
	if err != nil {
		return nil, err
	}
	connection, err := ovirtsdk.NewConnectionBuilder().
		URL(ovirtConfig.URL).
		Username(ovirtConfig.Username).
		Password(ovirtConfig.Password).
		CAFile(ovirtConfig.CAFile).Build()
	if err != nil {
		return nil, err
	}

	return connection, nil

}

var defaultOvirtConfigEnvVar = "OVIRT_CONFIG"
var defaultOvirtConfigPath = filepath.Join(os.Getenv("HOME"), ".ovirt", "ovirt-config.yaml")

// ErrCanNotLoadOvirtConfig is returned when the config file fails to load.
var ErrCanNotLoadOvirtConfig error = errors.New("can not load ovirt config")

// Config holds oVirt api access details.
type Config struct {
	URL      string `yaml:"ovirt_url"`
	Username string `yaml:"ovirt_username"`
	Password string `yaml:"ovirt_password"`
	CAFile   string `yaml:"ovirt_cafile,omitempty"`
}

// LoadOvirtConfig from the following location (first wins):
// 1. OVIRT_CONFIG env variable
// 2  $defaultOvirtConfigPath
func LoadOvirtConfig() ([]byte, error) {
	data, err := ioutil.ReadFile(discoverPath())
	if err != nil {
		return nil, err
	}
	return data, nil
}

// GetOvirtConfig will return an Config by loading
// the configuration from locations specified in @LoadOvirtConfig
// error is return if the configuration could not be retained.
func GetOvirtConfig() (*Config, error) {
	c := Config{}
	in, err := LoadOvirtConfig()
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(in, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func discoverPath() string {
	path, _ := os.LookupEnv(defaultOvirtConfigEnvVar)
	if path != "" {
		return path
	}

	return defaultOvirtConfigPath
}

// Save will serialize the config back into the locations
// specified in @LoadOvirtConfig, first location with a file, wins.
func (c *Config) Save() error {
	out, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	path := discoverPath()
	return ioutil.WriteFile(path, out, os.FileMode(0600))
}
