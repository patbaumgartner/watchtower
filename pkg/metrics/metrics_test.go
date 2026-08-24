package metrics_test

import (
	"errors"
	"testing"
	"time"

	"github.com/patbaumgartner/watchtower/internal/actions/mocks"
	"github.com/patbaumgartner/watchtower/pkg/metrics"
	"github.com/patbaumgartner/watchtower/pkg/session"
	"github.com/patbaumgartner/watchtower/pkg/types"
	"github.com/prometheus/client_golang/prometheus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMetrics(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metrics Suite")
}

// metricValue reads a gauge or counter from the default registry without extra deps
func metricValue(name string) float64 {
	families, err := prometheus.DefaultGatherer.Gather()
	Expect(err).NotTo(HaveOccurred())
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		series := family.GetMetric()
		if len(series) == 0 {
			break
		}
		if gauge := series[0].GetGauge(); gauge != nil {
			return gauge.GetValue()
		}
		if counter := series[0].GetCounter(); counter != nil {
			return counter.GetValue()
		}
	}
	return -1
}

func testReport() types.Report {
	progress := session.Progress{}

	fresh := mocks.CreateMockContainer("c-fresh", "/fresh", "img-fresh:latest", time.Now())
	progress.AddScanned(fresh, fresh.SafeImageID())

	updated := mocks.CreateMockContainer("c-updated", "/updated", "img-updated:latest", time.Now())
	progress.AddScanned(updated, "new-image")
	progress.MarkForUpdate(updated.ID())

	stale := mocks.CreateMockContainer("c-stale", "/stale", "img-stale:latest", time.Now())
	progress.AddScanned(stale, "new-image-stale")

	failed := mocks.CreateMockContainer("c-failed", "/failed", "img-failed:latest", time.Now())
	progress.AddScanned(failed, "new-image-failed")
	progress.MarkForUpdate(failed.ID())
	progress.UpdateFailed(map[types.ContainerID]error{failed.ID(): errors.New("boom")})

	return progress.Report()
}

var _ = Describe("the metrics handler", func() {
	It("should count report categories in NewMetric", func() {
		metric := metrics.NewMetric(testReport())
		Expect(metric.Scanned).To(Equal(4))
		// Updated counts updated and stale containers for backwards compatibility
		Expect(metric.Updated).To(Equal(2))
		Expect(metric.Failed).To(Equal(1))
	})

	It("should return the same handler from Default", func() {
		Expect(metrics.Default()).To(BeIdenticalTo(metrics.Default()))
	})

	It("should update the gauges from a registered scan", func() {
		metrics.RegisterScan(metrics.NewMetric(testReport()))
		Eventually(func() float64 { return metricValue("watchtower_containers_scanned") }).Should(Equal(4.0))
		Expect(metricValue("watchtower_containers_updated")).To(Equal(2.0))
		Expect(metricValue("watchtower_containers_failed")).To(Equal(1.0))
		Expect(metricValue("watchtower_scans_total")).To(BeNumerically(">=", 1.0))
	})

	It("should count a nil metric as a skipped scan and zero the gauges", func() {
		skippedBefore := metricValue("watchtower_scans_skipped")
		totalBefore := metricValue("watchtower_scans_total")
		metrics.RegisterScan(nil)
		Eventually(func() float64 { return metricValue("watchtower_scans_skipped") }).Should(Equal(skippedBefore + 1))
		Expect(metricValue("watchtower_scans_total")).To(Equal(totalBefore + 1))
		Expect(metricValue("watchtower_containers_scanned")).To(Equal(0.0))
		Expect(metricValue("watchtower_containers_updated")).To(Equal(0.0))
		Expect(metricValue("watchtower_containers_failed")).To(Equal(0.0))
	})
})
