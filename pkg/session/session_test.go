package session_test

import (
	"errors"
	"testing"
	"time"

	"github.com/patbaumgartner/watchtower/internal/actions/mocks"
	"github.com/patbaumgartner/watchtower/pkg/session"
	"github.com/patbaumgartner/watchtower/pkg/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSession(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Session Suite")
}

func newSessionContainer(id string, name string, image string) types.Container {
	return mocks.CreateMockContainer(id, name, image, time.Now())
}

var _ = Describe("session progress", func() {
	It("should bucket containers into the correct report categories", func() {
		progress := session.Progress{}

		skipped := newSessionContainer("c-skipped", "/skipped", "img-skipped:latest")
		progress.AddSkipped(skipped, errors.New("skip reason"))

		fresh := newSessionContainer("c-fresh", "/fresh", "img-fresh:latest")
		progress.AddScanned(fresh, fresh.SafeImageID())

		updated := newSessionContainer("c-updated", "/updated", "img-updated:latest")
		progress.AddScanned(updated, "new-image-updated")
		progress.MarkForUpdate(updated.ID())

		failed := newSessionContainer("c-failed", "/failed", "img-failed:latest")
		progress.AddScanned(failed, "new-image-failed")
		progress.MarkForUpdate(failed.ID())
		progress.UpdateFailed(map[types.ContainerID]error{failed.ID(): errors.New("boom")})

		stale := newSessionContainer("c-stale", "/stale", "img-stale:latest")
		progress.AddScanned(stale, "new-image-stale")

		report := progress.Report()

		names := func(reports []types.ContainerReport) []string {
			result := make([]string, 0, len(reports))
			for _, r := range reports {
				result = append(result, r.Name())
			}
			return result
		}

		Expect(names(report.Skipped())).To(ConsistOf("/skipped"))
		Expect(names(report.Fresh())).To(ConsistOf("/fresh"))
		Expect(names(report.Updated())).To(ConsistOf("/updated"))
		Expect(names(report.Failed())).To(ConsistOf("/failed"))
		Expect(names(report.Stale())).To(ConsistOf("/stale"))
		// Scanned includes everything that was not skipped
		Expect(names(report.Scanned())).To(ConsistOf("/fresh", "/updated", "/failed", "/stale"))
	})

	It("should expose container status fields through the report", func() {
		progress := session.Progress{}
		cont := newSessionContainer("c-status", "/status", "img-status:latest")
		progress.AddScanned(cont, "new-image")
		progress.MarkForUpdate(cont.ID())

		entry := progress.Report().Updated()[0]
		Expect(entry.ID()).To(Equal(cont.ID()))
		Expect(entry.Name()).To(Equal("/status"))
		Expect(entry.ImageName()).To(Equal("img-status:latest"))
		Expect(entry.CurrentImageID()).To(Equal(cont.SafeImageID()))
		Expect(entry.LatestImageID()).To(Equal(types.ImageID("new-image")))
		Expect(entry.State()).To(Equal("Updated"))
		Expect(entry.Error()).To(BeEmpty())
	})

	It("should expose errors for skipped and failed containers", func() {
		progress := session.Progress{}
		skipped := newSessionContainer("c-skipped", "/skipped", "img:latest")
		progress.AddSkipped(skipped, errors.New("skip reason"))

		entry := progress.Report().Skipped()[0]
		Expect(entry.State()).To(Equal("Skipped"))
		Expect(entry.Error()).To(Equal("skip reason"))
	})

	It("should list every container exactly once in All, sorted by name", func() {
		progress := session.Progress{}
		for _, name := range []string{"/bravo", "/alpha", "/charlie"} {
			cont := newSessionContainer("c"+name, name, "img"+name+":latest")
			progress.AddScanned(cont, types.ImageID("new-image"+name))
		}
		all := progress.Report().All()
		Expect(all).To(HaveLen(3))
		Expect(all[0].Name()).To(Equal("/alpha"))
		Expect(all[1].Name()).To(Equal("/bravo"))
		Expect(all[2].Name()).To(Equal("/charlie"))
	})
})
