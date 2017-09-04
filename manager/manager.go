package manager

import (
	"github.com/rancher/go-rancher/v3"
	"golang.org/x/sync/syncmap"
	"net/url"
)

type Manager struct {
	watchClients       syncmap.Map
	clientsets         syncmap.Map
	rancherClient      *client.RancherClient
	clusterOverrideURL string
}

func New(rancherClient *client.RancherClient) *Manager {
	m := &Manager{
		watchClients:  syncmap.Map{},
		clientsets:    syncmap.Map{},
		rancherClient: rancherClient,
	}

	if m.rancherClient != nil {
		u, _ := url.Parse(m.rancherClient.GetOpts().Url)
		u.Path = "/k8s/clusters/"
		m.clusterOverrideURL = u.String()
	}

	return m
}
