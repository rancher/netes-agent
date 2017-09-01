package manager

import (
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/netes-agent/watch"
	"k8s.io/client-go/kubernetes"
	"net/url"
)

type Manager struct {
	watchClients       map[string]*watch.Client
	clientsets         map[string]*kubernetes.Clientset
	rancherClient      *client.RancherClient
	clusterOverrideURL string
}

func New(rancherClient *client.RancherClient) *Manager {
	m := &Manager{
		watchClients:  make(map[string]*watch.Client),
		clientsets:    make(map[string]*kubernetes.Clientset),
		rancherClient: rancherClient,
	}

	if m.rancherClient != nil {
		u, _ := url.Parse(m.rancherClient.GetOpts().Url)
		u.Path = "/k8s/clusters/"
		m.clusterOverrideURL = u.String()
	}

	return m
}
