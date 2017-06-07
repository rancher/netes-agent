package types

import "github.com/rancher/go-rancher/v2"

type DeploymentUnit struct {
	Id                  string
	Uuid                string
	HostId              string
	Host                client.Host
	Containers          []client.Container
	RevisionId          string
	RequestedRevisionId string
	RevisionConfig      client.Service
}
