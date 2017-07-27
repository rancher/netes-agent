package main

import (
	"fmt"
	"os"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/rancher/go-rancher/v2"
	"github.com/rancherlabs/kattle/sync"
	"github.com/rancherlabs/kattle/types"
	"github.com/rancherlabs/kattle/watch"
	"github.com/urfave/cli"
)

const (
	cattleURLEnv          = "CATTLE_URL"
	cattleURLAccessKeyEnv = "CATTLE_ACCESS_KEY"
	cattleURLSecretKeyEnv = "CATTLE_SECRET_KEY"
)

var (
	deploymentUnitsCache []types.DeploymentUnit
	volumesCache         []types.Volume
)

var VERSION = "v0.0.0-dev"

func main() {
	app := cli.NewApp()
	app.Name = "kattle"
	app.Version = VERSION
	app.Usage = "You need help!"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name: "kubernetes-master",
		},
		cli.StringFlag{
			Name: "username",
		},
		cli.StringFlag{
			Name: "password",
		},
		cli.StringFlag{
			Name: "token",
		},
	}
	app.Action = action
	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}

func action(c *cli.Context) error {
	rancherClient, err := createRancherClient()
	if err != nil {
		return err
	}

	kubernetesURL := c.String("kubernetes-master")
	username := c.String("username")
	password := c.String("password")
	token := c.String("token")
	clientset, err := createKubernetesClient(kubernetesURL, username, password, token)
	if err != nil {
		return err
	}

	watchClient := watch.NewClient(rancherClient, clientset)
	watchClient.Start()

	time.Sleep(5 * time.Second)

	for {
		if err := updateDeploymentUnits(rancherClient); err != nil {
			fmt.Printf("Failed to update deployment units: %v", err)
		}
		if err := updateVolumes(rancherClient); err != nil {
			fmt.Printf("Failed to update volumes: %v", err)
		}
		if err = sync.Sync(clientset, watchClient, deploymentUnitsCache, volumesCache); err != nil {
			return err
		}
		time.Sleep(2 * time.Second)
	}
}

func createRancherClient() (*client.RancherClient, error) {
	url, err := client.NormalizeUrl(os.Getenv(cattleURLEnv))
	if err != nil {
		return nil, err
	}
	return client.NewRancherClient(&client.ClientOpts{
		Url:       url,
		AccessKey: os.Getenv(cattleURLAccessKeyEnv),
		SecretKey: os.Getenv(cattleURLSecretKeyEnv),
	})
}

func createKubernetesClient(url, username, password, token string) (*kubernetes.Clientset, error) {
	return kubernetes.NewForConfig(&rest.Config{
		Host:        url,
		Username:    username,
		Password:    password,
		BearerToken: token,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
	})
}

func updateDeploymentUnits(rancherClient *client.RancherClient) error {
	var newDeploymentUnitsCache []types.DeploymentUnit

	deploymentUnits, err := rancherClient.DeploymentUnit.List(&client.ListOpts{})
	if err != nil {
		return err
	}
	containers, err := rancherClient.Container.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"state": "running",
		},
	})
	if err != nil {
		return err
	}

	containersMap := map[string][]client.Container{}
	for _, container := range containers.Data {
		if _, ok := container.Labels["io.rancher.kattle"]; !ok {
			continue
		}

		if _, ok := containersMap[container.DeploymentUnitId]; ok {
			containersMap[container.DeploymentUnitId] = append(containersMap[container.DeploymentUnitId], container)
		} else {
			containersMap[container.DeploymentUnitId] = []client.Container{
				container,
			}
		}
	}

	for _, deploymentUnit := range deploymentUnits.Data {
		deploymentUnitContainers, ok := containersMap[deploymentUnit.Id]
		if !ok {
			continue
		}

		revision, err := rancherClient.Revision.ById(deploymentUnit.RevisionId)
		if err != nil {
			return err
		}

		deploymentUnit := types.DeploymentUnit{
			DeploymentUnit: deploymentUnit,
			Containers:     deploymentUnitContainers,
		}
		if revision != nil {
			deploymentUnit.Revision = *revision
		}

		newDeploymentUnitsCache = append(newDeploymentUnitsCache, deploymentUnit)
	}

	deploymentUnitsCache = newDeploymentUnitsCache

	return nil
}

func updateVolumes(rancherClient *client.RancherClient) error {
	volumes, err := rancherClient.Volume.List(&client.ListOpts{
		Filters: map[string]interface{}{},
	})
	if err != nil {
		return err
	}

	if volumes != nil {
		var metadataVolumes []types.Volume
		for _, volume := range volumes.Data {
			metadataVolumes = append(metadataVolumes, types.Volume{
				Volume: volume,
			})
		}
		volumesCache = metadataVolumes
	}

	return nil
}
