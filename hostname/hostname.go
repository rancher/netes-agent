package hostname

import (
	"fmt"
	"strings"

	"github.com/rancher/go-rancher/v2"
)

var (
	RancherClient *client.RancherClient
	// TODO: not thread-safe
	nameToUuid map[string]string
)

func UuidFromName(name string) (string, error) {
	uuid, ok := nameToUuid[name]
	if ok {
		return uuid, nil
	}

	hosts, err := RancherClient.Host.List(&client.ListOpts{})
	if err != nil {
		return "", err
	}

	for _, host := range hosts.Data {
		shortHostName := strings.Split(host.Hostname, ".")[0]
		nameToUuid[shortHostName] = host.Uuid
	}

	uuid, ok = nameToUuid[name]
	if ok {
		return uuid, nil
	}

	return "", fmt.Errorf("Failed to find host with name %s", name)
}
