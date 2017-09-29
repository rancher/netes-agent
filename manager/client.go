package manager

import "github.com/rancher/go-rancher/v3"

type Client interface {
	GetOpts() client.ClientOpts
	Publish(*client.Publish) error
	GetCluster(string) (*client.Cluster, error)
}

var _ Client = (*RancherClient)(nil)

type RancherClient struct {
	rancherClient *client.RancherClient
}

func NewRancherClient(client *client.RancherClient) *RancherClient {
	return &RancherClient{
		rancherClient: client,
	}
}

func (c *RancherClient) GetOpts() client.ClientOpts {
	return c.GetOpts()
}

func (c *RancherClient) Publish(pubish *client.Publish) error {
	_, err := c.rancherClient.Publish.Create(pubish)
	return err
}

func (c *RancherClient) GetCluster(id string) (*client.Cluster, error) {
	return c.rancherClient.Cluster.ById(id)
}
