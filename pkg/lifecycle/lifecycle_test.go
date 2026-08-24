package lifecycle_test

import (
	"errors"
	"testing"
	"time"

	dockerContainer "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/patbaumgartner/watchtower/internal/actions/mocks"
	"github.com/patbaumgartner/watchtower/pkg/lifecycle"
	"github.com/patbaumgartner/watchtower/pkg/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestLifecycle(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Lifecycle Suite")
}

// fakeClient implements container.Client with scriptable failures for hook paths
type fakeClient struct {
	containers   []types.Container
	listError    error
	getError     error
	execError    error
	skipUpdate   bool
	executed     []string
	execTimeouts []int
}

func (c *fakeClient) ListContainers(_ types.Filter) ([]types.Container, error) {
	return c.containers, c.listError
}

func (c *fakeClient) GetContainer(_ types.ContainerID) (types.Container, error) {
	if c.getError != nil {
		return nil, c.getError
	}
	return c.containers[0], nil
}

func (c *fakeClient) StopContainer(_ types.Container, _ time.Duration) error { return nil }

func (c *fakeClient) StartContainer(_ types.Container) (types.ContainerID, error) { return "", nil }

func (c *fakeClient) RenameContainer(_ types.Container, _ string) error { return nil }

func (c *fakeClient) IsContainerStale(_ types.Container, _ types.UpdateParams) (bool, types.ImageID, error) {
	return false, "", nil
}

func (c *fakeClient) ExecuteCommand(_ types.ContainerID, command string, timeout int) (bool, error) {
	c.executed = append(c.executed, command)
	c.execTimeouts = append(c.execTimeouts, timeout)
	return c.skipUpdate, c.execError
}

func (c *fakeClient) RemoveImageByID(_ types.ImageID) error { return nil }

func (c *fakeClient) WarnOnHeadPullFailed(_ types.Container) bool { return false }

func newHookContainer(running bool, labels map[string]string) types.Container {
	return mocks.CreateMockContainerWithConfig(
		"lifecycle-container",
		"/lifecycle-container",
		"lifecycle-image:latest",
		running,
		false,
		time.Now(),
		&dockerContainer.Config{
			Labels:       labels,
			ExposedPorts: network.PortSet{},
		})
}

var _ = Describe("lifecycle hooks", func() {
	Describe("ExecutePreChecks", func() {
		It("should run the pre-check command for every container", func() {
			client := &fakeClient{containers: []types.Container{
				newHookContainer(true, map[string]string{"com.centurylinklabs.watchtower.lifecycle.pre-check": "/pre-check-1.sh"}),
				newHookContainer(true, map[string]string{"com.centurylinklabs.watchtower.lifecycle.pre-check": "/pre-check-2.sh"}),
			}}
			lifecycle.ExecutePreChecks(client, types.UpdateParams{})
			Expect(client.executed).To(Equal([]string{"/pre-check-1.sh", "/pre-check-2.sh"}))
		})
		It("should do nothing when listing containers fails", func() {
			client := &fakeClient{listError: errors.New("list failed")}
			lifecycle.ExecutePreChecks(client, types.UpdateParams{})
			Expect(client.executed).To(BeEmpty())
		})
	})

	Describe("ExecutePostChecks", func() {
		It("should run the post-check command for every container", func() {
			client := &fakeClient{containers: []types.Container{
				newHookContainer(true, map[string]string{"com.centurylinklabs.watchtower.lifecycle.post-check": "/post-check.sh"}),
			}}
			lifecycle.ExecutePostChecks(client, types.UpdateParams{})
			Expect(client.executed).To(Equal([]string{"/post-check.sh"}))
		})
		It("should do nothing when listing containers fails", func() {
			client := &fakeClient{listError: errors.New("list failed")}
			lifecycle.ExecutePostChecks(client, types.UpdateParams{})
			Expect(client.executed).To(BeEmpty())
		})
	})

	Describe("ExecutePreCheckCommand", func() {
		It("should skip containers without a pre-check command", func() {
			client := &fakeClient{}
			lifecycle.ExecutePreCheckCommand(client, newHookContainer(true, map[string]string{}))
			Expect(client.executed).To(BeEmpty())
		})
		It("should log but not fail when the command errors", func() {
			client := &fakeClient{execError: errors.New("exec failed")}
			lifecycle.ExecutePreCheckCommand(client, newHookContainer(true, map[string]string{
				"com.centurylinklabs.watchtower.lifecycle.pre-check": "/pre-check.sh",
			}))
			Expect(client.executed).To(Equal([]string{"/pre-check.sh"}))
		})
	})

	Describe("ExecutePostCheckCommand", func() {
		It("should skip containers without a post-check command", func() {
			client := &fakeClient{}
			lifecycle.ExecutePostCheckCommand(client, newHookContainer(true, map[string]string{}))
			Expect(client.executed).To(BeEmpty())
		})
		It("should log but not fail when the command errors", func() {
			client := &fakeClient{execError: errors.New("exec failed")}
			lifecycle.ExecutePostCheckCommand(client, newHookContainer(true, map[string]string{
				"com.centurylinklabs.watchtower.lifecycle.post-check": "/post-check.sh",
			}))
			Expect(client.executed).To(Equal([]string{"/post-check.sh"}))
		})
	})

	Describe("ExecutePreUpdateCommand", func() {
		It("should skip containers without a pre-update command", func() {
			client := &fakeClient{}
			skip, err := lifecycle.ExecutePreUpdateCommand(client, newHookContainer(true, map[string]string{}))
			Expect(err).NotTo(HaveOccurred())
			Expect(skip).To(BeFalse())
			Expect(client.executed).To(BeEmpty())
		})
		It("should skip containers that are not running", func() {
			client := &fakeClient{}
			skip, err := lifecycle.ExecutePreUpdateCommand(client, newHookContainer(false, map[string]string{
				"com.centurylinklabs.watchtower.lifecycle.pre-update": "/pre-update.sh",
			}))
			Expect(err).NotTo(HaveOccurred())
			Expect(skip).To(BeFalse())
			Expect(client.executed).To(BeEmpty())
		})
		It("should pass the timeout label to the command execution", func() {
			client := &fakeClient{}
			_, err := lifecycle.ExecutePreUpdateCommand(client, newHookContainer(true, map[string]string{
				"com.centurylinklabs.watchtower.lifecycle.pre-update":         "/pre-update.sh",
				"com.centurylinklabs.watchtower.lifecycle.pre-update-timeout": "190",
			}))
			Expect(err).NotTo(HaveOccurred())
			Expect(client.executed).To(Equal([]string{"/pre-update.sh"}))
			Expect(client.execTimeouts).To(Equal([]int{190}))
		})
		It("should propagate the skip-update result and errors", func() {
			client := &fakeClient{skipUpdate: true, execError: errors.New("exec failed")}
			skip, err := lifecycle.ExecutePreUpdateCommand(client, newHookContainer(true, map[string]string{
				"com.centurylinklabs.watchtower.lifecycle.pre-update": "/pre-update.sh",
			}))
			Expect(err).To(MatchError("exec failed"))
			Expect(skip).To(BeTrue())
		})
	})

	Describe("ExecutePostUpdateCommand", func() {
		It("should do nothing when the new container cannot be fetched", func() {
			client := &fakeClient{getError: errors.New("get failed")}
			lifecycle.ExecutePostUpdateCommand(client, "some-id")
			Expect(client.executed).To(BeEmpty())
		})
		It("should skip containers without a post-update command", func() {
			client := &fakeClient{containers: []types.Container{newHookContainer(true, map[string]string{})}}
			lifecycle.ExecutePostUpdateCommand(client, "some-id")
			Expect(client.executed).To(BeEmpty())
		})
		It("should run the post-update command and tolerate errors", func() {
			client := &fakeClient{
				execError: errors.New("exec failed"),
				containers: []types.Container{newHookContainer(true, map[string]string{
					"com.centurylinklabs.watchtower.lifecycle.post-update":         "/post-update.sh",
					"com.centurylinklabs.watchtower.lifecycle.post-update-timeout": "3",
				})},
			}
			lifecycle.ExecutePostUpdateCommand(client, "some-id")
			Expect(client.executed).To(Equal([]string{"/post-update.sh"}))
			Expect(client.execTimeouts).To(Equal([]int{3}))
		})
	})
})
