package container

import (
	"encoding/json"
	"os"

	"github.com/moby/moby/api/types/network"
	"time"

	"github.com/patbaumgartner/watchtower/pkg/container/mocks"
	"github.com/patbaumgartner/watchtower/pkg/filters"
	t "github.com/patbaumgartner/watchtower/pkg/types"

	dockerContainer "github.com/moby/moby/api/types/container"

	cerrdefs "github.com/containerd/errdefs"
	cli "github.com/moby/moby/client"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/ghttp"
	"github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	gt "github.com/onsi/gomega/types"

	"context"
	"net/http"
)

var testHardwareAddress = network.HardwareAddr{0x02, 0x42, 0xac, 0x11, 0x00, 0x02}

var _ = Describe("the client", func() {
	var docker *cli.Client
	var mockServer *ghttp.Server
	BeforeEach(func() {
		mockServer = ghttp.NewServer()
		docker, _ = cli.NewClientWithOpts(
			cli.WithHost(mockServer.URL()),
			cli.WithHTTPClient(mockServer.HTTPTestServer.Client()),
			cli.WithAPIVersion("1.42"))
	})
	AfterEach(func() {
		mockServer.Close()
	})
	Describe("WarnOnHeadPullFailed", func() {
		containerUnknown := MockContainer(WithImageName("unknown.repo/prefix/imagename:latest"))
		containerKnown := MockContainer(WithImageName("docker.io/prefix/imagename:latest"))

		When(`warn on head failure is set to "always"`, func() {
			c := dockerClient{ClientOptions: ClientOptions{WarnOnHeadFailed: WarnAlways}}
			It("should always return true", func() {
				Expect(c.WarnOnHeadPullFailed(containerUnknown)).To(BeTrue())
				Expect(c.WarnOnHeadPullFailed(containerKnown)).To(BeTrue())
			})
		})
		When(`warn on head failure is set to "auto"`, func() {
			c := dockerClient{ClientOptions: ClientOptions{WarnOnHeadFailed: WarnAuto}}
			It("should return false for unknown repos", func() {
				Expect(c.WarnOnHeadPullFailed(containerUnknown)).To(BeFalse())
			})
			It("should return true for known repos", func() {
				Expect(c.WarnOnHeadPullFailed(containerKnown)).To(BeTrue())
			})
		})
		When(`warn on head failure is set to "never"`, func() {
			c := dockerClient{ClientOptions: ClientOptions{WarnOnHeadFailed: WarnNever}}
			It("should never return true", func() {
				Expect(c.WarnOnHeadPullFailed(containerUnknown)).To(BeFalse())
				Expect(c.WarnOnHeadPullFailed(containerKnown)).To(BeFalse())
			})
		})
	})
	When("pulling the latest image", func() {
		When("the image consist of a pinned hash", func() {
			It("should gracefully fail with a useful message", func() {
				c := dockerClient{}
				pinnedContainer := MockContainer(WithImageName("sha256:fa5269854a5e615e51a72b17ad3fd1e01268f278a6684c8ed3c5f0cdce3f230b"))
				err := c.PullImage(context.Background(), pinnedContainer)
				Expect(err).To(MatchError(`container uses a pinned image, and cannot be updated by watchtower`))
			})
		})
	})
	When("removing a running container", func() {
		When("the container still exist after stopping", func() {
			It("should attempt to remove the container", func() {
				container := MockContainer(WithContainerState(dockerContainer.State{Running: true}))
				containerStopped := MockContainer(WithContainerState(dockerContainer.State{Running: false}))

				cid := container.ContainerInfo().ID
				mockServer.AppendHandlers(
					mocks.KillContainerHandler(cid, mocks.Found),
					mocks.GetContainerHandler(cid, containerStopped.ContainerInfo()),
					mocks.RemoveContainerHandler(cid, mocks.Found),
					mocks.GetContainerHandler(cid, nil),
				)

				Expect(dockerClient{api: docker}.StopContainer(container, time.Minute)).To(Succeed())
			})
		})
		When("the container does not exist after stopping", func() {
			It("should not cause an error", func() {
				container := MockContainer(WithContainerState(dockerContainer.State{Running: true}))

				cid := container.ContainerInfo().ID
				mockServer.AppendHandlers(
					mocks.KillContainerHandler(cid, mocks.Found),
					mocks.GetContainerHandler(cid, nil),
					mocks.RemoveContainerHandler(cid, mocks.Missing),
				)

				Expect(dockerClient{api: docker}.StopContainer(container, time.Minute)).To(Succeed())
			})
		})
	})
	When("removing a image", func() {
		When("debug logging is enabled", func() {
			It("should log removed and untagged images", func() {
				imageA := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
				imageAParent := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
				images := map[string][]string{imageA: {imageAParent}}
				mockServer.AppendHandlers(mocks.RemoveImageHandler(images))
				c := dockerClient{api: docker}

				resetLogrus, logbuf := captureLogrus(logrus.DebugLevel)
				defer resetLogrus()

				Expect(c.RemoveImageByID(t.ImageID(imageA))).To(Succeed())

				shortA := t.ImageID(imageA).ShortID()
				shortAParent := t.ImageID(imageAParent).ShortID()

				Eventually(logbuf).Should(gbytes.Say(`deleted="%v, %v" untagged="?%v"?`, shortA, shortAParent, shortA))
			})
		})
		When("image is not found", func() {
			It("should return an error", func() {
				image := "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
				mockServer.AppendHandlers(mocks.RemoveImageHandler(nil))
				c := dockerClient{api: docker}

				err := c.RemoveImageByID(t.ImageID(image))
				Expect(cerrdefs.IsNotFound(err)).To(BeTrue())
			})
		})
	})
	Describe("API 1.42 container recreation", func() {
		It("should preserve AutoRemove and StopTimeout in the create request", func() {
			api42, err := cli.New(
				cli.WithHost(mockServer.URL()),
				cli.WithHTTPClient(mockServer.HTTPTestServer.Client()),
				cli.WithAPIVersion("1.42"),
			)
			Expect(err).NotTo(HaveOccurred())

			stopTimeout := 10
			container := MockContainer(WithImageName("docker.io/prefix/imagename:latest"))
			container.containerInfo.State = &dockerContainer.State{}
			container.containerInfo.Config.StopTimeout = &stopTimeout
			container.containerInfo.HostConfig.AutoRemove = true
			container.containerInfo.NetworkSettings = &dockerContainer.NetworkSettings{
				Networks: map[string]*network.EndpointSettings{},
			}

			mockServer.AppendHandlers(func(response http.ResponseWriter, request *http.Request) {
				Expect(request.URL.Path).To(HaveSuffix("/containers/create"))
				var createRequest dockerContainer.CreateRequest
				Expect(json.NewDecoder(request.Body).Decode(&createRequest)).To(Succeed())
				Expect(createRequest.HostConfig.AutoRemove).To(BeTrue())
				Expect(createRequest.Config.StopTimeout).NotTo(BeNil())
				Expect(*createRequest.Config.StopTimeout).To(Equal(stopTimeout))
				response.Header().Set("Content-Type", "application/json")
				response.WriteHeader(http.StatusCreated)
				Expect(json.NewEncoder(response).Encode(dockerContainer.CreateResponse{ID: "new-container"})).To(Succeed())
			})

			client := dockerClient{api: api42}
			_, err = client.StartContainer(container)
			Expect(err).NotTo(HaveOccurred())
		})
	})
	When("listing containers", func() {
		It("carries raw legacy inspect fields into recreation preflight", func() {
			mockServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, HaveSuffix("/containers/legacy/json")),
					ghttp.RespondWith(http.StatusOK, []byte(`{
						"Id":"legacy","Image":"image","Name":"/legacy","State":{"Running":true},
						"Config":{"Image":"example:latest","Labels":{},"MacAddress":"02:42:ac:11:00:02"},
						"HostConfig":{"NetworkMode":"default","PortBindings":{},"KernelMemory":1024,"KernelMemoryTCP":2048},
						"NetworkSettings":{"Networks":{}}
					}`)),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, HaveSuffix("/images/image/json")),
					ghttp.RespondWith(http.StatusOK, []byte(`{"Id":"image","Config":{}}`)),
				),
			)

			inspected, err := (dockerClient{api: docker}).GetContainer("legacy")
			Expect(err).NotTo(HaveOccurred())
			concrete, ok := inspected.(*Container)
			Expect(ok).To(BeTrue())
			Expect(concrete.legacyConfig).To(Equal(legacyContainerConfig{
				macAddress:      "02:42:ac:11:00:02",
				kernelMemory:    1024,
				kernelMemoryTCP: 2048,
			}))
			Expect(inspected.VerifyConfiguration()).To(MatchError(ContainSubstring("Docker API 1.44")))
		})
		When("no filter is provided", func() {
			It("should return all available containers", func() {
				mockServer.AppendHandlers(mocks.ListContainersHandler("running"))
				mockServer.AppendHandlers(mocks.GetContainerHandlers(&mocks.Watchtower, &mocks.Running)...)
				client := dockerClient{
					api:           docker,
					ClientOptions: ClientOptions{},
				}
				containers, err := client.ListContainers(filters.NoFilter)
				Expect(err).NotTo(HaveOccurred())
				Expect(containers).To(HaveLen(2))
			})
		})
		When("a filter matching nothing", func() {
			It("should return an empty array", func() {
				mockServer.AppendHandlers(mocks.ListContainersHandler("running"))
				mockServer.AppendHandlers(mocks.GetContainerHandlers(&mocks.Watchtower, &mocks.Running)...)
				filter := filters.FilterByNames([]string{"lollercoaster"}, filters.NoFilter)
				client := dockerClient{
					api:           docker,
					ClientOptions: ClientOptions{},
				}
				containers, err := client.ListContainers(filter)
				Expect(err).NotTo(HaveOccurred())
				Expect(containers).To(BeEmpty())
			})
		})
		When("a watchtower filter is provided", func() {
			It("should return only the watchtower container", func() {
				mockServer.AppendHandlers(mocks.ListContainersHandler("running"))
				mockServer.AppendHandlers(mocks.GetContainerHandlers(&mocks.Watchtower, &mocks.Running)...)
				client := dockerClient{
					api:           docker,
					ClientOptions: ClientOptions{},
				}
				containers, err := client.ListContainers(filters.WatchtowerContainersFilter)
				Expect(err).NotTo(HaveOccurred())
				Expect(containers).To(ConsistOf(withContainerImageName(Equal("patbaumgartner/watchtower:latest"))))
			})
		})
		When(`include stopped is enabled`, func() {
			It("should return both stopped and running containers", func() {
				mockServer.AppendHandlers(mocks.ListContainersHandler("running", "exited", "created"))
				mockServer.AppendHandlers(mocks.GetContainerHandlers(&mocks.Stopped, &mocks.Watchtower, &mocks.Running)...)
				client := dockerClient{
					api:           docker,
					ClientOptions: ClientOptions{IncludeStopped: true},
				}
				containers, err := client.ListContainers(filters.NoFilter)
				Expect(err).NotTo(HaveOccurred())
				Expect(containers).To(ContainElement(havingRunningState(false)))
			})
		})
		When(`include restarting is enabled`, func() {
			It("should return both restarting and running containers", func() {
				mockServer.AppendHandlers(mocks.ListContainersHandler("running", "restarting"))
				mockServer.AppendHandlers(mocks.GetContainerHandlers(&mocks.Watchtower, &mocks.Running, &mocks.Restarting)...)
				client := dockerClient{
					api:           docker,
					ClientOptions: ClientOptions{IncludeRestarting: true},
				}
				containers, err := client.ListContainers(filters.NoFilter)
				Expect(err).NotTo(HaveOccurred())
				Expect(containers).To(ContainElement(havingRestartingState(true)))
			})
		})
		When(`include restarting is disabled`, func() {
			It("should not return restarting containers", func() {
				mockServer.AppendHandlers(mocks.ListContainersHandler("running"))
				mockServer.AppendHandlers(mocks.GetContainerHandlers(&mocks.Watchtower, &mocks.Running)...)
				client := dockerClient{
					api:           docker,
					ClientOptions: ClientOptions{IncludeRestarting: false},
				}
				containers, err := client.ListContainers(filters.NoFilter)
				Expect(err).NotTo(HaveOccurred())
				Expect(containers).NotTo(ContainElement(havingRestartingState(true)))
			})
		})
		When(`a container uses container network mode`, func() {
			When(`the network container can be resolved`, func() {
				It("should return the container name instead of the ID", func() {
					consumerContainerRef := mocks.NetConsumerOK
					mockServer.AppendHandlers(mocks.GetContainerHandlers(&consumerContainerRef)...)
					client := dockerClient{
						api:           docker,
						ClientOptions: ClientOptions{},
					}
					container, err := client.GetContainer(consumerContainerRef.ContainerID())
					Expect(err).NotTo(HaveOccurred())
					networkMode := container.ContainerInfo().HostConfig.NetworkMode
					Expect(networkMode.ConnectedContainer()).To(Equal(mocks.NetSupplierContainerName))
				})
			})
			When(`the network container cannot be resolved`, func() {
				It("should still return the container ID", func() {
					consumerContainerRef := mocks.NetConsumerInvalidSupplier
					mockServer.AppendHandlers(mocks.GetContainerHandlers(&consumerContainerRef)...)
					client := dockerClient{
						api:           docker,
						ClientOptions: ClientOptions{},
					}
					container, err := client.GetContainer(consumerContainerRef.ContainerID())
					Expect(err).NotTo(HaveOccurred())
					networkMode := container.ContainerInfo().HostConfig.NetworkMode
					Expect(networkMode.ConnectedContainer()).To(Equal(mocks.NetSupplierNotFoundID))
				})
			})
		})
	})
	Describe(`ExecuteCommand`, func() {
		When(`logging`, func() {
			It("should include container id field when attach fails", func() {
				client := dockerClient{
					api:           docker,
					ClientOptions: ClientOptions{},
				}

				// Capture logrus output in buffer
				resetLogrus, logbuf := captureLogrus(logrus.DebugLevel)
				defer resetLogrus()

				user := ""
				containerID := t.ContainerID("ex-cont-id")
				execID := "ex-exec-id"
				cmd := "exec-cmd"

				mockServer.AppendHandlers(
					// API.ContainerExecCreate
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", HaveSuffix("containers/%v/exec", containerID)),
						ghttp.VerifyJSONRepresenting(dockerContainer.ExecCreateRequest{
							User: user,
							Tty:  true,
							Cmd: []string{
								"sh",
								"-c",
								cmd,
							},
						}),
						ghttp.RespondWithJSONEncoded(http.StatusOK, dockerContainer.ExecCreateResponse{ID: execID}),
					),
					// API.ContainerExecAttach starts the exec. A regular HTTP response
					// intentionally makes the mock fail the connection upgrade.
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", HaveSuffix("exec/%v/start", execID)),
						ghttp.VerifyJSONRepresenting(dockerContainer.ExecStartRequest{
							Detach: false,
							Tty:    true,
						}),
						ghttp.RespondWith(http.StatusOK, nil),
					),
				)

				_, err := client.ExecuteCommand(containerID, cmd, 1)
				Expect(err).To(HaveOccurred())
				Eventually(logbuf).Should(gbytes.Say(`containerID="?ex-cont-id"?`))
			})
		})
	})
	Describe(`GetNetworkConfig`, func() {
		When(`providing a container with network aliases`, func() {
			It(`should omit the container ID alias`, func() {
				client := dockerClient{
					api:           docker,
					ClientOptions: ClientOptions{IncludeRestarting: false},
				}
				container := MockContainer(WithImageName("docker.io/prefix/imagename:latest"))

				aliases := []string{"One", "Two", container.ID().ShortID(), "Four"}
				endpoints := map[string]*network.EndpointSettings{
					`test`: {Aliases: aliases, MacAddress: testHardwareAddress},
				}
				container.containerInfo.NetworkSettings = &dockerContainer.NetworkSettings{Networks: endpoints}
				Expect(container.ContainerInfo().NetworkSettings.Networks[`test`].Aliases).To(Equal(aliases))
				Expect(client.GetNetworkConfig(container).EndpointsConfig[`test`].Aliases).To(Equal([]string{"One", "Two", "Four"}))
				Expect(client.GetNetworkConfig(container).EndpointsConfig[`test`].MacAddress).To(BeEmpty())
				Expect(container.ContainerInfo().NetworkSettings.Networks[`test`].MacAddress).To(Equal(testHardwareAddress))
			})
		})
		When(`a compatible API is explicitly selected`, func() {
			It(`should preserve endpoint MAC addresses`, func() {
				original, present := os.LookupEnv("DOCKER_API_VERSION")
				Expect(os.Setenv("DOCKER_API_VERSION", "1.44")).To(Succeed())
				defer func() {
					if present {
						_ = os.Setenv("DOCKER_API_VERSION", original)
					} else {
						_ = os.Unsetenv("DOCKER_API_VERSION")
					}
				}()
				client := dockerClient{}
				container := MockContainer(WithImageName("docker.io/prefix/imagename:latest"))
				container.containerInfo.NetworkSettings = &dockerContainer.NetworkSettings{Networks: map[string]*network.EndpointSettings{
					`test`: {MacAddress: testHardwareAddress},
				}}

				Expect(client.GetNetworkConfig(container).EndpointsConfig[`test`].MacAddress).To(Equal(testHardwareAddress))
			})
			It(`should clear the running watchtower MAC to avoid a self-update collision`, func() {
				original, present := os.LookupEnv("DOCKER_API_VERSION")
				Expect(os.Setenv("DOCKER_API_VERSION", "1.44")).To(Succeed())
				defer func() {
					if present {
						_ = os.Setenv("DOCKER_API_VERSION", original)
					} else {
						_ = os.Unsetenv("DOCKER_API_VERSION")
					}
				}()
				client := dockerClient{}
				container := MockContainer(
					WithImageName("docker.io/patbaumgartner/watchtower:main"),
					WithLabels(map[string]string{"com.centurylinklabs.watchtower": "true"}),
				)
				container.containerInfo.NetworkSettings = &dockerContainer.NetworkSettings{Networks: map[string]*network.EndpointSettings{
					`test`: {MacAddress: testHardwareAddress},
				}}

				Expect(client.GetNetworkConfig(container).EndpointsConfig[`test`].MacAddress).To(BeEmpty())
			})
		})
	})
})

// Capture logrus output in buffer
func captureLogrus(level logrus.Level) (func(), *gbytes.Buffer) {

	logbuf := gbytes.NewBuffer()

	origOut := logrus.StandardLogger().Out
	logrus.SetOutput(logbuf)

	origLev := logrus.StandardLogger().Level
	logrus.SetLevel(level)

	return func() {
		logrus.SetOutput(origOut)
		logrus.SetLevel(origLev)
	}, logbuf
}

// Gomega matcher helpers

func withContainerImageName(matcher gt.GomegaMatcher) gt.GomegaMatcher {
	return WithTransform(containerImageName, matcher)
}

func containerImageName(container t.Container) string {
	return container.ImageName()
}

func havingRestartingState(expected bool) gt.GomegaMatcher {
	return WithTransform(func(container t.Container) bool {
		return container.ContainerInfo().State.Restarting
	}, Equal(expected))
}

func havingRunningState(expected bool) gt.GomegaMatcher {
	return WithTransform(func(container t.Container) bool {
		return container.ContainerInfo().State.Running
	}, Equal(expected))
}
