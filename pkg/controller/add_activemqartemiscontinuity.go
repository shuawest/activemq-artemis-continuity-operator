package controller

import (
	"github.com/rh-messaging/activemq-artemis-continuity-operator/pkg/controller/activemqartemiscontinuity"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, activemqartemiscontinuity.Add)
}
