package main

import (
	"os"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/rancher/go-rancher-metadata/metadata"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/kattle/sync"
	"github.com/urfave/cli"
)

const (
	cattleURLEnv          = "CATTLE_URL"
	cattleURLAccessKeyEnv = "CATTLE_ACCESS_KEY"
	cattleURLSecretKeyEnv = "CATTLE_SECRET_KEY"
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
			Name: "metadata-url",
		},
	}
	app.Action = action
	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}

func action(c *cli.Context) error {
	/*rancherClient, err := createRancherClient()
	if err != nil {
		return err
	}*/

	kubernetesURL := c.String("kubernetes-master")
	username := c.String("username")
	password := c.String("password")
	clientset, err := createKubernetesClient(kubernetesURL, username, password)
	if err != nil {
		return err
	}

	metadataURL := c.String("metadata-url")
	m := metadata.NewClient(metadataURL)

	//return m.OnChangeWithError(5, func(_ string) {
	/*if err := sync.Sync(m, rancherClient, kubernernetesClient); err != nil {
		logrus.Errorf("Sync failed: %v", err)
	}*/
	//})

	for {
		deploymentUnits, err := m.GetDeploymentUnits()
		if err != nil {
			return err
		}
		volumes, err := m.GetVolumes()
		if err != nil {
			return err
		}
		volumeDrivers, err := m.GetVolumeTemplates()
		if err != nil {
			return err
		}
		storageDrivers, err := m.GetStorageDrivers()
		if err != nil {
			return err
		}
		if err = sync.Sync(clientset, deploymentUnits, volumes, volumeDrivers, storageDrivers); err != nil {
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

func createKubernetesClient(url, username, password string) (*kubernetes.Clientset, error) {
	return kubernetes.NewForConfig(&rest.Config{
		Host:     url,
		Username: username,
		Password: password,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
	})
}
