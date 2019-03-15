package controller

import (
	"github.com/teiid/teiid-operator/pkg/controller/virtualdatabase"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, virtualdatabase.Add)
}
