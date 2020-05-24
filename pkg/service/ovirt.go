package service

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	ovirtsdk "github.com/ovirt/go-ovirt"
	"gopkg.in/yaml.v2"
)

type OvirtClient struct {
	connection *ovirtsdk.Connection
}

func NewOvirtClient() (*OvirtClient, error) {
	con, err := newOvirtConnection()
	if err != nil {
		return nil, err
	}
	return &OvirtClient{connection: con}, nil
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
		CAFile(ovirtConfig.CAFile).
		Insecure(ovirtConfig.Insecure).
		Build()
	if err != nil {
		return nil, err
	}

	return connection, nil

}

// GetConnection validates the connection we have is valid and either
// returns it, or creates a new one in case the connection isn't present
// or it was invalidated.
func (o *OvirtClient) GetConnection() (*ovirtsdk.Connection, error) {
	if o.connection == nil || o.connection.Test() != nil {
		return newOvirtConnection()
	}

	return o.connection, nil
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
	Insecure bool   `yaml:"ovirt_insecure,omitempty"`
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
