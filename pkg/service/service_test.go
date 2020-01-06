package service_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/ovirt/go-ovirt"

	. "ovirt/csi-driver/pkg/service"
)

type MockOvirtClient struct{}

func (m *MockOvirtClient) SystemService() ovirtsdk.SystemService {
	service := ovirtsdk.SystemService{}
}

var _ = Describe("Service", func() {
	var (
		ovirtClient MockOvirtClient
	)
	Describe("Controller Service", func() {
		It("creates a volume", func() {})
	})
})
