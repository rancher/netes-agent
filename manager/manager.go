package manager

import (
	"github.com/rancher/netes-agent/watch"
	"k8s.io/client-go/kubernetes"
)

type Manager struct {
	watchClients    map[string]*watch.Client
	clientsets      map[string]*kubernetes.Clientset
	cattleURL       string
	cattleAccessKey string
	cattleSecretKey string

	// TODO: remove once requests have cluster info
	currentClusterId string
}

func NewManager(cattleURL, cattleAccessKey, cattleSecretKey string) *Manager {
	return &Manager{
		watchClients:    make(map[string]*watch.Client),
		clientsets:      make(map[string]*kubernetes.Clientset),
		cattleURL:       cattleURL,
		cattleAccessKey: cattleAccessKey,
		cattleSecretKey: cattleSecretKey,
	}
}
