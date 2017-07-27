package types

import (
	"github.com/rancher/go-rancher/v2"
)

type DeploymentUnit struct {
	client.DeploymentUnit
	Containers []client.Container
	Revision   client.Revision
	Host       client.Host
}

type Volume struct {
	client.Volume
	Metadata map[string]interface{}
}
