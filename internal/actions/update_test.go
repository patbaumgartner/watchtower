package actions_test

import (
	"time"

	dockerContainer "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/api/types/network"
	"github.com/patbaumgartner/watchtower/internal/actions"
	"github.com/patbaumgartner/watchtower/pkg/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/patbaumgartner/watchtower/internal/actions/mocks"
)

func getCommonTestData(keepContainer string) *TestData {
	return &TestData{
		NameOfContainerToKeep: keepContainer,
		Containers: []types.Container{
			CreateMockContainer(
				"test-container-01",
				"test-container-01",
				"fake-image:latest",
				time.Now().AddDate(0, 0, -1)),
			CreateMockContainer(
				"test-container-02",
				"test-container-02",
				"fake-image:latest",
				time.Now()),
			CreateMockContainer(
				"test-container-02",
				"test-container-02",
				"fake-image:latest",
				time.Now()),
		},
	}
}

func getLinkedTestData(withImageInfo bool) *TestData {
	staleContainer := CreateMockContainer(
		"test-container-01",
		"/test-container-01",
		"fake-image1:latest",
		time.Now().AddDate(0, 0, -1))

	var imageInfo *image.InspectResponse
	if withImageInfo {
		imageInfo = CreateMockImageInfo("test-container-02")
	}
	linkingContainer := CreateMockContainerWithLinks(
		"test-container-02",
		"/test-container-02",
		"fake-image2:latest",
		time.Now(),
		[]string{staleContainer.Name()},
		imageInfo)

	return &TestData{
		Staleness: map[string]bool{linkingContainer.Name(): false},
		Containers: []types.Container{
			staleContainer,
			linkingContainer,
		},
	}
}

var _ = Describe("the update action", func() {
	When("watchtower has been instructed to clean up", func() {
		When("there are multiple containers using the same image", func() {
			It("should only try to remove the image once", func() {
				client := CreateMockClient(getCommonTestData(""), false, false)
				_, err := actions.Update(client, types.UpdateParams{Cleanup: true})
				Expect(err).NotTo(HaveOccurred())
				Expect(client.TestData.TriedToRemoveImageCount).To(Equal(1))
			})
		})
		When("there are multiple containers using different images", func() {
			It("should try to remove each of them", func() {
				testData := getCommonTestData("")
				testData.Containers = append(
					testData.Containers,
					CreateMockContainer(
						"unique-test-container",
						"unique-test-container",
						"unique-fake-image:latest",
						time.Now(),
					),
				)
				client := CreateMockClient(testData, false, false)
				_, err := actions.Update(client, types.UpdateParams{Cleanup: true})
				Expect(err).NotTo(HaveOccurred())
				Expect(client.TestData.TriedToRemoveImageCount).To(Equal(2))
			})
		})
		When("there are linked containers being updated", func() {
			It("should not try to remove their images", func() {
				client := CreateMockClient(getLinkedTestData(true), false, false)
				_, err := actions.Update(client, types.UpdateParams{Cleanup: true})
				Expect(err).NotTo(HaveOccurred())
				Expect(client.TestData.TriedToRemoveImageCount).To(Equal(1))
			})
		})
		When("performing a rolling restart update", func() {
			It("should try to remove the image once", func() {
				client := CreateMockClient(getCommonTestData(""), false, false)
				_, err := actions.Update(client, types.UpdateParams{Cleanup: true, RollingRestart: true})
				Expect(err).NotTo(HaveOccurred())
				Expect(client.TestData.TriedToRemoveImageCount).To(Equal(1))
			})
		})
		When("updating a linked container with missing image info", func() {
			It("should skip the restart set before stopping containers", func() {
				client := CreateMockClient(getLinkedTestData(false), false, false)

				report, err := actions.Update(client, types.UpdateParams{})
				Expect(err).NotTo(HaveOccurred())
				Expect(client.TestData.TriedToStopCount).To(BeZero())
				Expect(client.TestData.TriedToStartCount).To(BeZero())
				Expect(report.Skipped()).To(HaveLen(1))
			})
		})
		When("a dependency-restarted container uses configuration unsupported by Docker API 1.42", func() {
			It("should not stop any container in the update set", func() {
				testData := getLinkedTestData(true)
				linked := testData.Containers[1]
				linked.ContainerInfo().HostConfig.Mounts = []mount.Mount{{
					Type:        mount.TypeBind,
					Target:      "/data",
					BindOptions: &mount.BindOptions{ReadOnlyForceRecursive: true},
				}}
				client := CreateMockClient(testData, false, false)

				report, err := actions.Update(client, types.UpdateParams{})

				Expect(err).NotTo(HaveOccurred())
				Expect(client.TestData.TriedToStopCount).To(BeZero())
				Expect(client.TestData.TriedToStartCount).To(BeZero())
				Expect(report.Skipped()).To(HaveLen(1))
			})
			It("should still run the post-check hook when preflight aborts", func() {
				testData := getLinkedTestData(true)
				testData.Containers[0].ContainerInfo().Config.Labels["com.centurylinklabs.watchtower.lifecycle.pre-check"] = "/PreUpdateReturn0.sh"
				testData.Containers[0].ContainerInfo().Config.Labels["com.centurylinklabs.watchtower.lifecycle.post-check"] = "/PostCheck.sh"
				testData.Containers[1].ContainerInfo().HostConfig.Mounts = []mount.Mount{{
					Type:        mount.TypeBind,
					Target:      "/data",
					BindOptions: &mount.BindOptions{ReadOnlyForceRecursive: true},
				}}
				client := CreateMockClient(testData, false, false)
				Expect(testData.Containers[0].GetLifecyclePostCheckCommand()).To(Equal("/PostCheck.sh"))

				_, err := actions.Update(client, types.UpdateParams{LifecycleHooks: true})

				Expect(err).NotTo(HaveOccurred())
				Expect(testData.Containers[0].GetLifecyclePostCheckCommand()).To(Equal("/PostCheck.sh"))
				Expect(client.TestData.ExecutedCommands).To(ContainElement("/PreUpdateReturn0.sh"))
				Expect(client.TestData.ExecutedCommands).To(ContainElement("/PostCheck.sh"))
			})
		})
	})

	When("watchtower has been instructed to monitor only", func() {
		When("certain containers are set to monitor only", func() {
			It("should not update those containers", func() {
				client := CreateMockClient(
					&TestData{
						NameOfContainerToKeep: "test-container-02",
						Containers: []types.Container{
							CreateMockContainer(
								"test-container-01",
								"test-container-01",
								"fake-image1:latest",
								time.Now()),
							CreateMockContainerWithConfig(
								"test-container-02",
								"test-container-02",
								"fake-image2:latest",
								false,
								false,
								time.Now(),
								&dockerContainer.Config{
									Labels: map[string]string{
										"com.centurylinklabs.watchtower.monitor-only": "true",
									},
								}),
						},
					},
					false,
					false,
				)
				_, err := actions.Update(client, types.UpdateParams{Cleanup: true})
				Expect(err).NotTo(HaveOccurred())
				Expect(client.TestData.TriedToRemoveImageCount).To(Equal(1))
			})
			It("should not let an incompatible monitor-only container block eligible updates", func() {
				testData := &TestData{
					Containers: []types.Container{
						CreateMockContainer("eligible", "eligible", "eligible:latest", time.Now()),
						CreateMockContainerWithConfig(
							"monitor-only", "monitor-only", "monitor-only:latest", false, false, time.Now(),
							&dockerContainer.Config{Labels: map[string]string{"com.centurylinklabs.watchtower.monitor-only": "true"}},
						),
					},
				}
				testData.Containers[1].ContainerInfo().HostConfig.Mounts = []mount.Mount{{
					Type: mount.TypeImage, Target: "/data",
				}}
				client := CreateMockClient(testData, false, false)

				_, err := actions.Update(client, types.UpdateParams{})

				Expect(err).NotTo(HaveOccurred())
				Expect(client.TestData.TriedToStopCount).To(Equal(1))
				Expect(client.TestData.TriedToStartCount).To(Equal(1))
			})
		})

		When("monitor only is set globally", func() {
			It("should not update any containers", func() {
				client := CreateMockClient(
					&TestData{
						Containers: []types.Container{
							CreateMockContainer(
								"test-container-01",
								"test-container-01",
								"fake-image:latest",
								time.Now()),
							CreateMockContainer(
								"test-container-02",
								"test-container-02",
								"fake-image:latest",
								time.Now()),
						},
					},
					false,
					false,
				)
				_, err := actions.Update(client, types.UpdateParams{Cleanup: true, MonitorOnly: true})
				Expect(err).NotTo(HaveOccurred())
				Expect(client.TestData.TriedToRemoveImageCount).To(Equal(0))
			})
			When("watchtower has been instructed to have label take precedence", func() {
				It("it should update containers when monitor only is set to false", func() {
					client := CreateMockClient(
						&TestData{
							//NameOfContainerToKeep: "test-container-02",
							Containers: []types.Container{
								CreateMockContainerWithConfig(
									"test-container-02",
									"test-container-02",
									"fake-image2:latest",
									false,
									false,
									time.Now(),
									&dockerContainer.Config{
										Labels: map[string]string{
											"com.centurylinklabs.watchtower.monitor-only": "false",
										},
									}),
							},
						},
						false,
						false,
					)
					_, err := actions.Update(client, types.UpdateParams{Cleanup: true, MonitorOnly: true, LabelPrecedence: true})
					Expect(err).NotTo(HaveOccurred())
					Expect(client.TestData.TriedToRemoveImageCount).To(Equal(1))
				})
				It("it should update not containers when monitor only is set to true", func() {
					client := CreateMockClient(
						&TestData{
							//NameOfContainerToKeep: "test-container-02",
							Containers: []types.Container{
								CreateMockContainerWithConfig(
									"test-container-02",
									"test-container-02",
									"fake-image2:latest",
									false,
									false,
									time.Now(),
									&dockerContainer.Config{
										Labels: map[string]string{
											"com.centurylinklabs.watchtower.monitor-only": "true",
										},
									}),
							},
						},
						false,
						false,
					)
					_, err := actions.Update(client, types.UpdateParams{Cleanup: true, MonitorOnly: true, LabelPrecedence: true})
					Expect(err).NotTo(HaveOccurred())
					Expect(client.TestData.TriedToRemoveImageCount).To(Equal(0))
				})
				It("it should update not containers when monitor only is not set", func() {
					client := CreateMockClient(
						&TestData{
							Containers: []types.Container{
								CreateMockContainer(
									"test-container-01",
									"test-container-01",
									"fake-image:latest",
									time.Now()),
							},
						},
						false,
						false,
					)
					_, err := actions.Update(client, types.UpdateParams{Cleanup: true, MonitorOnly: true, LabelPrecedence: true})
					Expect(err).NotTo(HaveOccurred())
					Expect(client.TestData.TriedToRemoveImageCount).To(Equal(0))
				})

			})
		})
	})

	When("watchtower has been instructed to run lifecycle hooks", func() {

		When("pre-update script returns 1", func() {
			It("should not update those containers", func() {
				client := CreateMockClient(
					&TestData{
						//NameOfContainerToKeep: "test-container-02",
						Containers: []types.Container{
							CreateMockContainerWithConfig(
								"test-container-02",
								"test-container-02",
								"fake-image2:latest",
								true,
								false,
								time.Now(),
								&dockerContainer.Config{
									Labels: map[string]string{
										"com.centurylinklabs.watchtower.lifecycle.pre-update-timeout": "190",
										"com.centurylinklabs.watchtower.lifecycle.pre-update":         "/PreUpdateReturn1.sh",
									},
									ExposedPorts: network.PortSet{},
								}),
						},
					},
					false,
					false,
				)

				_, err := actions.Update(client, types.UpdateParams{Cleanup: true, LifecycleHooks: true})
				Expect(err).NotTo(HaveOccurred())
				Expect(client.TestData.TriedToRemoveImageCount).To(Equal(0))
			})

		})

		When("prupddate script returns 75", func() {
			It("should not update those containers", func() {
				client := CreateMockClient(
					&TestData{
						//NameOfContainerToKeep: "test-container-02",
						Containers: []types.Container{
							CreateMockContainerWithConfig(
								"test-container-02",
								"test-container-02",
								"fake-image2:latest",
								true,
								false,
								time.Now(),
								&dockerContainer.Config{
									Labels: map[string]string{
										"com.centurylinklabs.watchtower.lifecycle.pre-update-timeout": "190",
										"com.centurylinklabs.watchtower.lifecycle.pre-update":         "/PreUpdateReturn75.sh",
									},
									ExposedPorts: network.PortSet{},
								}),
						},
					},
					false,
					false,
				)
				_, err := actions.Update(client, types.UpdateParams{Cleanup: true, LifecycleHooks: true})
				Expect(err).NotTo(HaveOccurred())
				Expect(client.TestData.TriedToRemoveImageCount).To(Equal(0))
			})

		})

		When("prupddate script returns 0", func() {
			It("should update those containers", func() {
				client := CreateMockClient(
					&TestData{
						//NameOfContainerToKeep: "test-container-02",
						Containers: []types.Container{
							CreateMockContainerWithConfig(
								"test-container-02",
								"test-container-02",
								"fake-image2:latest",
								true,
								false,
								time.Now(),
								&dockerContainer.Config{
									Labels: map[string]string{
										"com.centurylinklabs.watchtower.lifecycle.pre-update-timeout": "190",
										"com.centurylinklabs.watchtower.lifecycle.pre-update":         "/PreUpdateReturn0.sh",
									},
									ExposedPorts: network.PortSet{},
								}),
						},
					},
					false,
					false,
				)
				_, err := actions.Update(client, types.UpdateParams{Cleanup: true, LifecycleHooks: true})
				Expect(err).NotTo(HaveOccurred())
				Expect(client.TestData.TriedToRemoveImageCount).To(Equal(1))
			})
		})

		When("container is linked to restarting containers", func() {
			It("should be marked for restart", func() {

				provider := CreateMockContainerWithConfig(
					"test-container-provider",
					"/test-container-provider",
					"fake-image2:latest",
					true,
					false,
					time.Now(),
					&dockerContainer.Config{
						Labels:       map[string]string{},
						ExposedPorts: network.PortSet{},
					})

				provider.SetStale(true)

				consumer := CreateMockContainerWithConfig(
					"test-container-consumer",
					"/test-container-consumer",
					"fake-image3:latest",
					true,
					false,
					time.Now(),
					&dockerContainer.Config{
						Labels: map[string]string{
							"com.centurylinklabs.watchtower.depends-on": "test-container-provider",
						},
						ExposedPorts: network.PortSet{},
					})

				containers := []types.Container{
					provider,
					consumer,
				}

				Expect(provider.ToRestart()).To(BeTrue())
				Expect(consumer.ToRestart()).To(BeFalse())

				actions.UpdateImplicitRestart(containers)

				Expect(containers[0].ToRestart()).To(BeTrue())
				Expect(containers[1].ToRestart()).To(BeTrue())

			})

		})

		When("container is not running", func() {
			It("skip running preupdate", func() {
				client := CreateMockClient(
					&TestData{
						//NameOfContainerToKeep: "test-container-02",
						Containers: []types.Container{
							CreateMockContainerWithConfig(
								"test-container-02",
								"test-container-02",
								"fake-image2:latest",
								false,
								false,
								time.Now(),
								&dockerContainer.Config{
									Labels: map[string]string{
										"com.centurylinklabs.watchtower.lifecycle.pre-update-timeout": "190",
										"com.centurylinklabs.watchtower.lifecycle.pre-update":         "/PreUpdateReturn1.sh",
									},
									ExposedPorts: network.PortSet{},
								}),
						},
					},
					false,
					false,
				)
				_, err := actions.Update(client, types.UpdateParams{Cleanup: true, LifecycleHooks: true})
				Expect(err).NotTo(HaveOccurred())
				Expect(client.TestData.TriedToRemoveImageCount).To(Equal(1))
			})

		})

		When("container is restarting", func() {
			It("skip running preupdate", func() {
				client := CreateMockClient(
					&TestData{
						//NameOfContainerToKeep: "test-container-02",
						Containers: []types.Container{
							CreateMockContainerWithConfig(
								"test-container-02",
								"test-container-02",
								"fake-image2:latest",
								false,
								true,
								time.Now(),
								&dockerContainer.Config{
									Labels: map[string]string{
										"com.centurylinklabs.watchtower.lifecycle.pre-update-timeout": "190",
										"com.centurylinklabs.watchtower.lifecycle.pre-update":         "/PreUpdateReturn1.sh",
									},
									ExposedPorts: network.PortSet{},
								}),
						},
					},
					false,
					false,
				)
				_, err := actions.Update(client, types.UpdateParams{Cleanup: true, LifecycleHooks: true})
				Expect(err).NotTo(HaveOccurred())
				Expect(client.TestData.TriedToRemoveImageCount).To(Equal(1))
			})

		})

	})
})
