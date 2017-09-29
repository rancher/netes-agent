package integration

import (
	"github.com/rancher/go-rancher/v3"
)

var (
	basicEvent   = getEvent("basic")
	podSpecEvent = modifyEvent(getEvent("basic"), func(_ *client.DeploymentSyncRequest, c *client.Container) {
		c.IpcMode = "host"
		c.PidMode = "host"
	})
	containerSpecEvent = modifyEvent(getEvent("basic"), func(_ *client.DeploymentSyncRequest, c *client.Container) {
		c.Tty = true
		c.StdinOpen = true
		c.WorkingDir = "/usr"
		c.Environment = map[string]string{
			"TEST": "true",
		}
	})
	securityContextEvent = modifyEvent(getEvent("basic"), func(_ *client.DeploymentSyncRequest, c *client.Container) {
		c.CapAdd = []string{"SYS_NICE"}
	})
	upgradeEvent = modifyEvent(getEvent("basic"), func(request *client.DeploymentSyncRequest, c *client.Container) {
		request.Revision = "newrevision"
		c.Image = "nginx:1.13"
		c.ImageUuid = "docker:nginx:1.13"
	})
	alternateNamespaceEvent = modifyEvent(getEvent("basic"), func(request *client.DeploymentSyncRequest, c *client.Container) {
		request.Namespace = "testnamespace"
	})
)
