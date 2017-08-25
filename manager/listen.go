package manager

import (
	"github.com/Sirupsen/logrus"
	"github.com/rancher/event-subscriber/events"
)

func (m *Manager) Listen(workerCount int) error {
	logrus.Infof("Listening for events on %s", m.cattleURL)

	pingConfig := events.PingConfig{
		SendPingInterval:  5000,
		CheckPongInterval: 5000,
		MaxPongWait:       60000,
	}
	router, err := events.NewEventRouter("", 0, m.cattleURL, m.cattleAccessKey, m.cattleSecretKey, nil, map[string]events.EventHandler{
		"external.compute.instance.activate": wrapHandler(m.HandleComputeInstanceActivate),
		"external.compute.instance.remove":   wrapHandler(m.HandleComputeInstanceRemove),
		"cluster.create":                     m.handleClusterCreateOrUpdate,
		"cluster.remove":                     m.handleClusterRemove,
		"cluster.update":                     m.handleClusterCreateOrUpdate,
	}, "", workerCount, pingConfig)
	if err != nil {
		return err
	}

	return router.StartWithoutCreate(nil)
}
