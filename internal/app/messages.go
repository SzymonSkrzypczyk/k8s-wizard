package app

import "github.com/k8s-wizard/internal/kubectl"

// Messages are custom events sent through the Bubble Tea update loop

// podNamesLoadedMsg is sent when pod names have been fetched
type podNamesLoadedMsg struct {
	names []string
	err   error
}

// commandExecutedMsg is sent when a kubectl command has been executed
type commandExecutedMsg struct {
	result kubectl.CommandResult
	err    error
}

// favouriteSavedMsg is sent when a favourite has been saved
type favouriteSavedMsg struct {
	err error
}

// favouriteDeletedMsg is sent when a favourite has been deleted
type favouriteDeletedMsg struct {
	err error
}
