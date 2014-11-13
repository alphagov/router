package integration

import (
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	vegeta "github.com/tsenart/vegeta/lib"
)

const routerLatencyThreshold = 20 * time.Millisecond

var _ = Describe("Performance", func() {

	Context("two healthy backends", func() {
		var (
			backend1 *httptest.Server
			backend2 *httptest.Server
		)

		BeforeEach(func() {
			backend1 = startSimpleBackend("backend 1")
			backend2 = startSimpleBackend("backend 2")
			addBackend("backend-1", backend1.URL)
			addBackend("backend-2", backend2.URL)
			addBackendRoute("/one", "backend-1")
			addBackendRoute("/two", "backend-2")
			reloadRoutes()
		})
		AfterEach(func() {
			backend1.Close()
			backend2.Close()
		})

		It("should not significantly increase latency", func() {
			assertPerformantRouter(backend1, backend2)
		})
	})

})

func assertPerformantRouter(backend1, backend2 *httptest.Server) {
	directResultsCh := startVegetaAttack([]string{backend1.URL + "/one", backend2.URL + "/two"})
	routerResultsCh := startVegetaAttack([]string{routerURL("/one"), routerURL("/two")})

	directResults := <-directResultsCh
	routerResults := <-routerResultsCh

	Expect(routerResults.Requests).To(Equal(directResults.Requests))
	Expect(routerResults.Success).To(Equal(1.0)) // 100% success rate
	Expect(directResults.Success).To(Equal(1.0)) // 100% success rate

	Expect(routerResults.Latencies.Mean).To(BeNumerically("~", directResults.Latencies.Mean, routerLatencyThreshold))
	Expect(routerResults.Latencies.P95).To(BeNumerically("~", directResults.Latencies.P95, routerLatencyThreshold))
	Expect(routerResults.Latencies.P99).To(BeNumerically("~", directResults.Latencies.P99, routerLatencyThreshold*2))
	Expect(routerResults.Latencies.Max).To(BeNumerically("~", directResults.Latencies.Max, routerLatencyThreshold*2))
}

func startVegetaAttack(targetURLs []string) chan *vegeta.Metrics {
	targets := make([]*vegeta.Target, 0, len(targetURLs))
	for _, url := range targetURLs {
		targets = append(targets, &vegeta.Target{
			Method: "GET",
			URL:    url,
		})
	}
	targeter := vegeta.NewStaticTargeter(targets...)
	metricsChan := make(chan *vegeta.Metrics, 1)
	go vegetaAttack(targeter, metricsChan)
	return metricsChan
}

func vegetaAttack(targeter vegeta.Targeter, metricsChan chan *vegeta.Metrics) {
	attacker := vegeta.NewAttacker()

	var results vegeta.Results
	for res := range attacker.Attack(targeter, 50, 10*time.Second) {
		results = append(results, res)
	}

	metrics := vegeta.NewMetrics(results)
	metricsChan <- metrics
}
