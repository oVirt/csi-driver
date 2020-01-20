package controller

import (
	"github.com/ovirt/csi-driver/pkg/controller/ovirtcsioperator"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, ovirtcsioperator.Add)
}
