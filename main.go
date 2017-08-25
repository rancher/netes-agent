package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/netes-agent/manager"
	"github.com/urfave/cli"
)

const (
	cattleURLEnv       = "CATTLE_URL"
	cattleAccessKeyEnv = "CATTLE_ACCESS_KEY"
	cattleSecretKeyEnv = "CATTLE_SECRET_KEY"
)

var VERSION = "v0.0.0-dev"

func main() {
	app := cli.NewApp()
	app.Name = "netes-agent"
	app.Version = VERSION
	app.Flags = []cli.Flag{}
	app.Action = action
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func action(c *cli.Context) error {
	cattleURL := os.Getenv(cattleURLEnv)
	cattleAccessKey := os.Getenv(cattleAccessKeyEnv)
	cattleSecretKey := os.Getenv(cattleSecretKeyEnv)

	rancherClient, err := client.NewRancherClient(&client.ClientOpts{
		Url:       cattleURL,
		AccessKey: cattleAccessKey,
		SecretKey: cattleSecretKey,
	})
	if err != nil {
		return err
	}

	clusters, err := rancherClient.Cluster.List(&client.ListOpts{})
	if err != nil {
		return err
	}

	manager := manager.NewManager(cattleURL, cattleAccessKey, cattleSecretKey)

	if err := manager.SyncClusters(clusters.Data); err != nil {
		return err
	}

	return manager.Listen(250)
}
