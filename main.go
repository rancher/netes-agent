package main

import (
	"os"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

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
			Name:  "kubernetes-master",
			Value: "http://165.227.2.161/",
		},
		cli.StringFlag{
			Name:  "metadata-url",
			Value: "http://rancher-metadata/2015-12-19",
		},
	}
	app.Action = action
	app.Run(os.Args)
}

func action(c *cli.Context) error {
	rancherClient, err := createRancherClient()
	if err != nil {
		return err
	}

	kubernetesURL := c.String("kubernetes-url")
	kubernernetesClient, err := createKubernetesClient(kubernetesURL)
	if err != nil {
		return err
	}

	/*m := metadata.NewClient(metadataURL)

	return m.OnChangeWithError(5, func(_ string) {
		if err := sync.Sync(m, rancherClient, kubernernetesClient); err != nil {
			logrus.Errorf("Sync failed: %v", err)
		}
	})*/

	for {
		if err = sync.Sync(nil, rancherClient, kubernernetesClient); err != nil {
			return err
		}
		time.Sleep(time.Second * 2)
	}

	return sync.Sync(nil, rancherClient, kubernernetesClient)
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

func createKubernetesClient(url string) (*kubernetes.Clientset, error) {
	return kubernetes.NewForConfig(&rest.Config{
		Host: url,
	})
}
