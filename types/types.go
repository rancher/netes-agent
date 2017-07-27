package types

import "github.com/rancher/go-rancher/v2"

type DeploymentUnit struct {
	client.DeploymentUnit
	Revision   string
	Containers []client.Container
	Host       client.Host
}

func (d *DeploymentUnit) Primary() client.Container {
	if len(d.Containers) > -0 {
		return d.Containers[0]
	}
	return client.Container{}
}

type Volume struct {
	client.Volume
	Metadata map[string]interface{}
}
