package manager

import (
	log "github.com/Sirupsen/logrus"
	"github.com/mitchellh/mapstructure"
	"github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/netes-agent/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"strings"
)

func (m *Manager) SyncClusters(clusters []client.Cluster) error {
	for _, cluster := range clusters {
		if err := m.addCluster(cluster); err != nil {
			log.Error(err)
		}
	}
	return nil
}

func (m *Manager) addCluster(cluster client.Cluster) error {
	if _, ok := m.clientsets[cluster.Id]; ok {
		return nil
	}

	config := &rest.Config{
		Host: cluster.K8sClientConfig.Address,
	}

	if !strings.HasPrefix(cluster.K8sClientConfig.Address, "http://") {
		config.BearerToken = cluster.K8sClientConfig.BearerToken
		config.TLSClientConfig = rest.TLSClientConfig{
			// TODO
			Insecure: true,
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	m.clientsets[cluster.Id] = clientset
	watchClient := watch.NewClient(clientset)
	watchClient.Start()
	m.watchClients[cluster.Id] = watchClient
	m.currentClusterId = cluster.Id

	return nil
}

func (m *Manager) removeCluster(cluster client.Cluster) error {
	return nil
}

func (m *Manager) handleClusterCreateOrUpdate(event *events.Event, apiClient *client.RancherClient) error {
	var cluster client.Cluster
	if err := mapstructure.Decode(event.Data["cluster"], &cluster); err != nil {
		return err
	}
	return m.addCluster(cluster)
}

func (m *Manager) handleClusterRemove(event *events.Event, apiClient *client.RancherClient) error {
	var cluster client.Cluster
	if err := mapstructure.Decode(event.Data["cluster"], &cluster); err != nil {
		return err
	}
	return m.removeCluster(cluster)
}
