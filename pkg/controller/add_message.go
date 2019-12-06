package controller

import (
	"github.com/pohsienshih/chatbot-operator/chatbot-operator/pkg/controller/message"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, message.Add)
}
