package sorter_test

import (
	"sort"
	"testing"
	"time"

	dockerContainer "github.com/moby/moby/api/types/container"
	"github.com/patbaumgartner/watchtower/pkg/container"
	"github.com/patbaumgartner/watchtower/pkg/sorter"
	"github.com/patbaumgartner/watchtower/pkg/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSorter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sorter Suite")
}

func newTestContainer(name string, created string, links []string) types.Container {
	return container.NewContainer(
		&dockerContainer.InspectResponse{
			ID:      name + "-id",
			Name:    name,
			Created: created,
			HostConfig: &dockerContainer.HostConfig{
				Links: links,
			},
			Config: &dockerContainer.Config{
				Labels: map[string]string{},
			},
		},
		nil,
	)
}

func rfc3339(t time.Time) string {
	return t.Format(time.RFC3339Nano)
}

var _ = Describe("the sorter", func() {
	Describe("ByCreated", func() {
		It("should sort containers by their created date", func() {
			now := time.Now()
			containers := []types.Container{
				newTestContainer("/newest", rfc3339(now), nil),
				newTestContainer("/oldest", rfc3339(now.AddDate(0, 0, -2)), nil),
				newTestContainer("/middle", rfc3339(now.AddDate(0, 0, -1)), nil),
			}
			sort.Sort(sorter.ByCreated(containers))
			Expect(containers[0].Name()).To(Equal("/oldest"))
			Expect(containers[1].Name()).To(Equal("/middle"))
			Expect(containers[2].Name()).To(Equal("/newest"))
		})
		It("should treat an unparsable created date as the current time", func() {
			containers := []types.Container{
				newTestContainer("/unparsable", "not-a-timestamp", nil),
				newTestContainer("/old", rfc3339(time.Now().AddDate(-5, 0, 0)), nil),
			}
			sort.Sort(sorter.ByCreated(containers))
			Expect(containers[0].Name()).To(Equal("/old"))
			Expect(containers[1].Name()).To(Equal("/unparsable"))
		})
	})

	Describe("SortByDependencies", func() {
		It("should sort linked containers after their dependencies", func() {
			now := time.Now()
			db := newTestContainer("/db", rfc3339(now), nil)
			app := newTestContainer("/app", rfc3339(now), []string{"/db:/app/db"})
			sorted, err := sorter.SortByDependencies([]types.Container{app, db})
			Expect(err).NotTo(HaveOccurred())
			Expect(sorted).To(HaveLen(2))
			Expect(sorted[0].Name()).To(Equal("/db"))
			Expect(sorted[1].Name()).To(Equal("/app"))
		})
		It("should keep unlinked containers in input order", func() {
			now := time.Now()
			sorted, err := sorter.SortByDependencies([]types.Container{
				newTestContainer("/one", rfc3339(now), nil),
				newTestContainer("/two", rfc3339(now), nil),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(sorted[0].Name()).To(Equal("/one"))
			Expect(sorted[1].Name()).To(Equal("/two"))
		})
		It("should return an error on circular references", func() {
			now := time.Now()
			a := newTestContainer("/a", rfc3339(now), []string{"/b:/a/b"})
			b := newTestContainer("/b", rfc3339(now), []string{"/a:/b/a"})
			_, err := sorter.SortByDependencies([]types.Container{a, b})
			Expect(err).To(MatchError(ContainSubstring("circular reference")))
		})
		It("should ignore links to containers that are not in the list", func() {
			now := time.Now()
			app := newTestContainer("/app", rfc3339(now), []string{"/missing:/app/missing"})
			sorted, err := sorter.SortByDependencies([]types.Container{app})
			Expect(err).NotTo(HaveOccurred())
			Expect(sorted).To(HaveLen(1))
		})
	})
})
