package manager

import (
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/mitchellh/mapstructure"
	"github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/netes-agent/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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

	if cluster.K8sClientConfig == nil {
		return nil
	}

	config := &rest.Config{
		Host:        m.getHost(cluster),
		BearerToken: cluster.K8sClientConfig.BearerToken,
	}

	if !strings.HasPrefix(cluster.K8sClientConfig.Address, "http://") {
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

	log.Infof("Registered cluster %s (%s) at %s", cluster.Name, cluster.Id, config.Host)

	return nil
}

func (m *Manager) getHost(cluster client.Cluster) string {
	if m.clusterOverrideURL != "" {
		return m.clusterOverrideURL + cluster.Id
	}
	if strings.HasSuffix(cluster.K8sClientConfig.Address, "443") {
		return "https://" + cluster.K8sClientConfig.Address
	}
	return "http://" + cluster.K8sClientConfig.Address
}

func (m *Manager) removeCluster(cluster client.Cluster) error {
	return nil
}

func (m *Manager) handleClusterCreateOrUpdate(event *events.Event) (*client.Publish, error) {
	var cluster client.Cluster
	if err := mapstructure.Decode(event.Data["cluster"], &cluster); err != nil {
		return nil, err
	}
	return emptyReply(event), m.addCluster(cluster)
}

func (m *Manager) handleClusterRemove(event *events.Event) (*client.Publish, error) {
	var cluster client.Cluster
	if err := mapstructure.Decode(event.Data["cluster"], &cluster); err != nil {
		return nil, err
	}
	return emptyReply(event), m.removeCluster(cluster)
}
