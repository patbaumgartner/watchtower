// Package container contains code related to dealing with docker containers
package container

import (
	"errors"
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/patbaumgartner/watchtower/internal/dockercompat"
	"github.com/patbaumgartner/watchtower/internal/util"
	wt "github.com/patbaumgartner/watchtower/pkg/types"
	"github.com/sirupsen/logrus"

	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
)

// NewContainer returns a new Container instance instantiated with the
// specified ContainerInfo and ImageInfo structs.
func NewContainer(containerInfo *dockercontainer.InspectResponse, imageInfo *image.InspectResponse) *Container {
	return &Container{
		containerInfo: containerInfo,
		imageInfo:     imageInfo,
	}
}

// Container represents a running Docker dockercontainer.
type Container struct {
	LinkedToRestarting bool
	Stale              bool

	containerInfo *dockercontainer.InspectResponse
	imageInfo     *image.InspectResponse
}

// IsLinkedToRestarting returns the current value of the LinkedToRestarting field for the container
func (c *Container) IsLinkedToRestarting() bool {
	return c.LinkedToRestarting
}

// IsStale returns the current value of the Stale field for the container
func (c *Container) IsStale() bool {
	return c.Stale
}

// SetLinkedToRestarting sets the LinkedToRestarting field for the container
func (c *Container) SetLinkedToRestarting(value bool) {
	c.LinkedToRestarting = value
}

// SetStale implements sets the Stale field for the container
func (c *Container) SetStale(value bool) {
	c.Stale = value
}

// ContainerInfo fetches JSON info for the container
func (c Container) ContainerInfo() *dockercontainer.InspectResponse {
	return c.containerInfo
}

// ID returns the Docker container ID.
func (c Container) ID() wt.ContainerID {
	return wt.ContainerID(c.containerInfo.ID)
}

// IsRunning returns a boolean flag indicating whether or not the current
// container is running. The status is determined by the value of the
// container's "State.Running" property.
func (c Container) IsRunning() bool {
	return c.containerInfo.State.Running
}

// IsRestarting returns a boolean flag indicating whether or not the current
// container is restarting. The status is determined by the value of the
// container's "State.Restarting" property.
func (c Container) IsRestarting() bool {
	return c.containerInfo.State.Restarting
}

// Name returns the Docker container name.
func (c Container) Name() string {
	return c.containerInfo.Name
}

// ImageID returns the ID of the Docker image that was used to start the
// dockercontainer. May cause nil dereference if imageInfo is not set!
func (c Container) ImageID() wt.ImageID {
	return wt.ImageID(c.imageInfo.ID)
}

// SafeImageID returns the ID of the Docker image that was used to start the container if available,
// otherwise returns an empty string
func (c Container) SafeImageID() wt.ImageID {
	if c.imageInfo == nil {
		return ""
	}
	return wt.ImageID(c.imageInfo.ID)
}

// ImageName returns the name of the Docker image that was used to start the
// dockercontainer. If the original image was specified without a particular tag, the
// "latest" tag is assumed.
func (c Container) ImageName() string {
	// Compatibility w/ Zodiac deployments
	imageName, ok := c.getLabelValue(zodiacLabel)
	if !ok {
		imageName = c.containerInfo.Config.Image
	}

	if !strings.Contains(imageName, ":") {
		imageName = fmt.Sprintf("%s:latest", imageName)
	}

	return imageName
}

// Enabled returns the value of the container enabled label and if the label
// was set.
func (c Container) Enabled() (bool, bool) {
	rawBool, ok := c.getLabelValue(enableLabel)
	if !ok {
		return false, false
	}

	parsedBool, err := strconv.ParseBool(rawBool)
	if err != nil {
		return false, false
	}

	return parsedBool, true
}

// IsMonitorOnly returns whether the container should only be monitored based on values of
// the monitor-only label, the monitor-only argument and the label-take-precedence argument.
func (c Container) IsMonitorOnly(params wt.UpdateParams) bool {
	return c.getContainerOrGlobalBool(params.MonitorOnly, monitorOnlyLabel, params.LabelPrecedence)
}

// IsNoPull returns whether the image should be pulled based on values of
// the no-pull label, the no-pull argument and the label-take-precedence argument.
func (c Container) IsNoPull(params wt.UpdateParams) bool {
	return c.getContainerOrGlobalBool(params.NoPull, noPullLabel, params.LabelPrecedence)
}

func (c Container) getContainerOrGlobalBool(globalVal bool, label string, contPrecedence bool) bool {
	if contVal, err := c.getBoolLabelValue(label); err != nil {
		if !errors.Is(err, errorLabelNotFound) {
			logrus.WithField("error", err).WithField("label", label).Warn("Failed to parse label value")
		}
		return globalVal
	} else {
		if contPrecedence {
			return contVal
		} else {
			return contVal || globalVal
		}
	}
}

// Scope returns the value of the scope UID label and if the label
// was set.
func (c Container) Scope() (string, bool) {
	rawString, ok := c.getLabelValue(scope)
	if !ok {
		return "", false
	}

	return rawString, true
}

// Links returns a list containing the names of all the containers to which
// this container is linked.
func (c Container) Links() []string {
	var links []string

	dependsOnLabelValue := c.getLabelValueOrEmpty(dependsOnLabel)

	if dependsOnLabelValue != "" {
		for _, link := range strings.Split(dependsOnLabelValue, ",") {
			// Since the container names need to start with '/', let's prepend it if it's missing
			if !strings.HasPrefix(link, "/") {
				link = "/" + link
			}
			links = append(links, link)
		}

		return links
	}

	if (c.containerInfo != nil) && (c.containerInfo.HostConfig != nil) {
		for _, link := range c.containerInfo.HostConfig.Links {
			name := strings.Split(link, ":")[0]
			links = append(links, name)
		}

		// If the container uses another container for networking, it can be considered an implicit link
		// since the container would stop working if the network supplier were to be recreated
		networkMode := c.containerInfo.HostConfig.NetworkMode
		if networkMode.IsContainer() {
			links = append(links, networkMode.ConnectedContainer())
		}
	}

	return links
}

// ToRestart return whether the container should be restarted, either because
// is stale or linked to another stale dockercontainer.
func (c Container) ToRestart() bool {
	return c.Stale || c.LinkedToRestarting
}

// IsWatchtower returns a boolean flag indicating whether or not the current
// container is the watchtower container itself. The watchtower container is
// identified by the presence of the "com.centurylinklabs.watchtower" label in
// the container metadata.
func (c Container) IsWatchtower() bool {
	return ContainsWatchtowerLabel(c.containerInfo.Config.Labels)
}

// PreUpdateTimeout checks whether a container has a specific timeout set
// for how long the pre-update command is allowed to run. This value is expressed
// either as an integer, in minutes, or as 0 which will allow the command/script
// to run indefinitely. Users should be cautious with the 0 option, as that
// could result in watchtower waiting forever.
func (c Container) PreUpdateTimeout() int {
	var minutes int
	var err error

	val := c.getLabelValueOrEmpty(preUpdateTimeoutLabel)

	minutes, err = strconv.Atoi(val)
	if err != nil || val == "" {
		return 1
	}

	return minutes
}

// PostUpdateTimeout checks whether a container has a specific timeout set
// for how long the post-update command is allowed to run. This value is expressed
// either as an integer, in minutes, or as 0 which will allow the command/script
// to run indefinitely. Users should be cautious with the 0 option, as that
// could result in watchtower waiting forever.
func (c Container) PostUpdateTimeout() int {
	var minutes int
	var err error

	val := c.getLabelValueOrEmpty(postUpdateTimeoutLabel)

	minutes, err = strconv.Atoi(val)
	if err != nil || val == "" {
		return 1
	}

	return minutes
}

// StopSignal returns the custom stop signal (if any) that is encoded in the
// container's metadata. If the container has not specified a custom stop
// signal, the empty string "" is returned.
func (c Container) StopSignal() string {
	return c.getLabelValueOrEmpty(signalLabel)
}

// GetCreateConfig returns the container's current Config converted into a format
// that can be re-submitted to the Docker create API.
//
// Ideally, we'd just be able to take the ContainerConfig from the old container
// and use it as the starting point for creating the new container; however,
// the ContainerConfig that comes back from the Inspect call merges the default
// configuration (the stuff specified in the metadata for the image itself)
// with the overridden configuration (the stuff that you might specify as part
// of the "docker run").
//
// In order to avoid unintentionally overriding the
// defaults in the new image we need to separate the override options from the
// default options. To do this we have to compare the ContainerConfig for the
// running container with the ContainerConfig from the image that container was
// started from. This function returns a ContainerConfig which contains just
// the options overridden at runtime.
func (c Container) GetCreateConfig() *dockercontainer.Config {
	config := *c.containerInfo.Config
	hostConfig := c.containerInfo.HostConfig
	imageConfig := c.imageInfo.Config
	if config.Healthcheck != nil {
		healthcheck := *config.Healthcheck
		healthcheck.Test = append([]string(nil), config.Healthcheck.Test...)
		config.Healthcheck = &healthcheck
	}
	config.ExposedPorts = maps.Clone(config.ExposedPorts)

	if config.WorkingDir == imageConfig.WorkingDir {
		config.WorkingDir = ""
	}

	if config.User == imageConfig.User {
		config.User = ""
	}

	if hostConfig.NetworkMode.IsContainer() {
		config.Hostname = ""
	}

	if util.SliceEqual(config.Entrypoint, imageConfig.Entrypoint) {
		config.Entrypoint = nil
		if util.SliceEqual(config.Cmd, imageConfig.Cmd) {
			config.Cmd = nil
		}
	}

	// Clear HEALTHCHECK configuration (if default)
	if config.Healthcheck != nil && imageConfig.Healthcheck != nil {
		if util.SliceEqual(config.Healthcheck.Test, imageConfig.Healthcheck.Test) {
			config.Healthcheck.Test = nil
		}

		if config.Healthcheck.Retries == imageConfig.Healthcheck.Retries {
			config.Healthcheck.Retries = 0
		}

		if config.Healthcheck.Interval == imageConfig.Healthcheck.Interval {
			config.Healthcheck.Interval = 0
		}

		if config.Healthcheck.Timeout == imageConfig.Healthcheck.Timeout {
			config.Healthcheck.Timeout = 0
		}

		if config.Healthcheck.StartPeriod == imageConfig.Healthcheck.StartPeriod {
			config.Healthcheck.StartPeriod = 0
		}

		if config.Healthcheck.StartInterval == imageConfig.Healthcheck.StartInterval {
			config.Healthcheck.StartInterval = 0
		}
	}

	config.Env = util.SliceSubtract(config.Env, imageConfig.Env)

	config.Labels = util.StringMapSubtract(config.Labels, imageConfig.Labels)

	config.Volumes = util.StructMapSubtract(config.Volumes, imageConfig.Volumes)

	// subtract ports exposed in image from container
	if config.ExposedPorts == nil {
		config.ExposedPorts = make(map[nat.Port]struct{})
	}
	for k := range config.ExposedPorts {
		if _, ok := imageConfig.ExposedPorts[string(k)]; ok {
			delete(config.ExposedPorts, k)
		}
	}
	for p := range c.containerInfo.HostConfig.PortBindings {
		config.ExposedPorts[p] = struct{}{}
	}

	config.Image = c.ImageName()
	return &config
}

// GetCreateHostConfig returns the container's current HostConfig with any links
// re-written so that they can be re-submitted to the Docker create API.
func (c Container) GetCreateHostConfig() *dockercontainer.HostConfig {
	hostConfig := *c.containerInfo.HostConfig
	hostConfig.Links = append([]string(nil), c.containerInfo.HostConfig.Links...)

	for i, link := range hostConfig.Links {
		name := link[0:strings.Index(link, ":")]
		alias := link[strings.LastIndex(link, "/"):]

		hostConfig.Links[i] = fmt.Sprintf("%s:%s", name, alias)
	}

	return &hostConfig
}

// HasImageInfo returns whether image information could be retrieved for the container
func (c Container) HasImageInfo() bool {
	return c.imageInfo != nil
}

// ImageInfo fetches the ImageInspect data of the current container
func (c Container) ImageInfo() *image.InspectResponse {
	return c.imageInfo
}

// VerifyConfiguration checks the container and image configurations for nil references to make sure
// that the container can be recreated once deleted
func (c Container) VerifyConfiguration() error {
	if c.imageInfo == nil {
		return errorNoImageInfo
	}

	containerInfo := c.ContainerInfo()
	if containerInfo == nil {
		return errorNoContainerInfo
	}

	containerConfig := containerInfo.Config
	if containerConfig == nil {
		return errorInvalidConfig
	}

	hostConfig := containerInfo.HostConfig
	if hostConfig == nil {
		return errorInvalidConfig
	}
	if containerConfig.Healthcheck != nil && containerConfig.Healthcheck.StartInterval != 0 && !dockercompat.Supports("1.44") {
		return fmt.Errorf("container healthcheck uses a start interval, which requires Docker API 1.44 or later")
	}
	if hostConfig.KernelMemory != 0 && (dockercompat.Supports("1.42") || !dockercompat.Supports("1.40")) {
		return fmt.Errorf("container configures kernel memory, which requires Docker API 1.40 or 1.41")
	}
	//lint:ignore SA1019 API 1.25 preflight must detect this field before Docker silently clears it.
	if hostConfig.KernelMemoryTCP != 0 && !dockercompat.Supports("1.40") {
		return fmt.Errorf("container configures TCP kernel memory, which requires Docker API 1.40 or later")
	}
	if hostConfig.ConsoleSize != [2]uint{} && !dockercompat.Supports("1.42") {
		return fmt.Errorf("container configures console size, which requires Docker API 1.42 or later")
	}
	if len(hostConfig.Annotations) != 0 && !dockercompat.Supports("1.43") {
		return fmt.Errorf("container configures annotations, which require Docker API 1.43 or later")
	}

	for _, containerMount := range hostConfig.Mounts {
		if containerMount.Type == mount.TypeBind && containerMount.ReadOnly && !dockercompat.Supports("1.44") &&
			(containerMount.BindOptions == nil || !containerMount.BindOptions.ReadOnlyNonRecursive) {
			return fmt.Errorf("container mount %q requires recursive read-only semantics, which require Docker API 1.44 or later", containerMount.Target)
		}
		if containerMount.BindOptions != nil {
			if containerMount.BindOptions.NonRecursive && !dockercompat.Supports("1.40") {
				return fmt.Errorf("container mount %q uses bind-recursive=disabled, which requires Docker API 1.40 or later", containerMount.Target)
			}
			if containerMount.BindOptions.CreateMountpoint && !dockercompat.Supports("1.42") {
				return fmt.Errorf("container mount %q uses bind create-mountpoint, which requires Docker API 1.42 or later", containerMount.Target)
			}
			if containerMount.BindOptions.ReadOnlyForceRecursive && !dockercompat.Supports("1.44") {
				return fmt.Errorf("container mount %q uses bind-recursive=readonly, which requires Docker API 1.44 or later", containerMount.Target)
			}
		}
		if containerMount.VolumeOptions != nil && containerMount.VolumeOptions.Subpath != "" && !dockercompat.Supports("1.45") {
			return fmt.Errorf("container mount %q uses a volume subpath, which requires Docker API 1.45 or later", containerMount.Target)
		}
		if containerMount.Type == mount.TypeImage && !dockercompat.Supports("1.48") {
			return fmt.Errorf("container mount %q uses an image mount, which requires Docker API 1.48 or later", containerMount.Target)
		}
	}
	if containerInfo.NetworkSettings != nil && !dockercompat.Supports("1.48") {
		for name, endpoint := range containerInfo.NetworkSettings.Networks {
			if endpoint != nil && endpoint.GwPriority != 0 {
				return fmt.Errorf("container network %q configures gateway priority, which requires Docker API 1.48 or later", name)
			}
		}
	}

	return nil
}
