package controller

import (
	"github.com/krsacme/multus-go-operator/pkg/controller/multus"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, multus.Add)
}
