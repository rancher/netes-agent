package types

import "github.com/rancher/go-rancher/v2"

type DeploymentUnit struct {
	Id                  string
	Uuid                string
	HostId              string
	Containers          []client.Container
	RevisionId          string
	RequestedRevisionId string
	RevisionConfig      client.Service
}
