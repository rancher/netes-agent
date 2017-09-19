package manager

import (
	"strings"

	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/netes-agent/utils"
	"github.com/rancher/netes-agent/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func (m *Manager) getCluster(clusterID string) (*kubernetes.Clientset, *watch.Client, error) {
	clientsetRaw, _ := m.clientsets.Load(clusterID)
	watchClientRaw, ok := m.watchClients.Load(clusterID)
	if ok {
		return clientsetRaw.(*kubernetes.Clientset), watchClientRaw.(*watch.Client), nil
	}
	cluster, err := m.rancherClient.Cluster.ById(clusterID)
	if err != nil {
		return nil, nil, err
	}
	if cluster.State == "removing" || cluster.Removed != "" {
		return nil, nil, fmt.Errorf("Cluster %s is removed or being removed", cluster.Name)
	}
	return m.addCluster(cluster)
}

func (m *Manager) addCluster(cluster *client.Cluster) (*kubernetes.Clientset, *watch.Client, error) {
	if cluster.K8sClientConfig == nil {
		return nil, nil, fmt.Errorf("Cluster %s is missing credentials", cluster.Name)
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
		return nil, nil, err
	}

	m.clientsets.Store(cluster.Id, clientset)
	watchClient := watch.NewClient(clientset)
	watchClient.Start()
	m.watchClients.Store(cluster.Id, watchClient)

	log.Infof("Registered cluster %s (%s) at %s", cluster.Name, cluster.Id, config.Host)

	return clientset, watchClient, nil
}

func (m *Manager) getHost(cluster *client.Cluster) string {
	if m.clusterOverrideURL != "" {
		return m.clusterOverrideURL + cluster.Id
	}
	if strings.HasSuffix(cluster.K8sClientConfig.Address, "443") {
		return "https://" + cluster.K8sClientConfig.Address
	}
	return "http://" + cluster.K8sClientConfig.Address
}

func (m *Manager) removeCluster(cluster client.Cluster) error {
	m.clientsets.Delete(cluster.Id)
	watchClient, ok := m.watchClients.Load(cluster.Id)
	if !ok {
		return nil
	}
	watchClient.(*watch.Client).Stop()
	m.watchClients.Delete(cluster.Id)
	return nil
}

func (m *Manager) handleClusterRemove(event *events.Event, apiClient *client.RancherClient) (*client.Publish, error) {
	var cluster client.Cluster
	if err := utils.ConvertByJSON(event.Data["cluster"], &cluster); err != nil {
		return nil, err
	}
	log.Infof("Removing cluster %s", cluster.Name)
	return emptyReply(event), m.removeCluster(cluster)
}
