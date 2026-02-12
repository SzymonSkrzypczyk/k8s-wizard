package app

import "github.com/k8s-wizard/internal/kubectl"

// Messages are custom events sent through the Bubble Tea update loop

// resourceNamesLoadedMsg is sent when resource names have been fetched for selection
type resourceNamesLoadedMsg struct {
	names []string
	err   error
}

// commandExecutedMsg is sent when a kubectl command has been executed
type commandExecutedMsg struct {
	result kubectl.CommandResult
	err    error
}

type commandHelpLoadedMsg struct {
	result kubectl.CommandResult
	err    error
}

type clusterConnectivityCheckedMsg struct {
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

// favouriteRenamedMsg is sent when a favourite has been renamed
type favouriteRenamedMsg struct {
	err error
}

type savedOutputRenamedMsg struct {
	err error
}

// outputSavedMsg is sent when command output has been saved to a file
type outputSavedMsg struct {
	filename string
	err      error
}

// savedOutputsLoadedMsg is sent when saved output files have been loaded
type savedOutputsLoadedMsg struct {
	files []string
	err   error
}
