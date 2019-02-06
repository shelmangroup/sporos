package controller

import (
	"github.com/shelmangroup/sporos/pkg/controller/sporos"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, sporos.Add)
}
