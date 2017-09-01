package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/netes-agent/manager"
	"github.com/urfave/cli"
)

var VERSION = "v0.0.0-dev"

func main() {
	app := cli.NewApp()
	app.Name = "netes-agent"
	app.Version = VERSION
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "access-key",
			EnvVar: "CATTLE_ACCESS_KEY",
			Usage:  "Rancher access key",
		},
		cli.StringFlag{
			Name:   "secret-key",
			EnvVar: "CATTLE_SECRET_KEY",
			Usage:  "Rancher secret key",
		},
		cli.StringFlag{
			Name:   "url",
			Value: "http://localhost:8080/v3",
			EnvVar: "CATTLE_URL",
			Usage:  "Rancher URL",
		},
	}
	app.Action = action
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func action(c *cli.Context) error {
	rancherClient, err := client.NewRancherClient(&client.ClientOpts{
		Url:       c.String("url"),
		AccessKey: os.Getenv("access-key"),
		SecretKey: os.Getenv("secret-key"),
	})
	if err != nil {
		return err
	}

	clusters, err := rancherClient.Cluster.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"removed_null": nil,
			"state_ne": "removing",
		},
	})
	if err != nil {
		return err
	}

	manager := manager.New(rancherClient)

	if err := manager.SyncClusters(clusters.Data); err != nil {
		return err
	}

	return manager.Listen()
}
