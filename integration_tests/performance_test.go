package integration

import (
	"net/http/httptest"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	vegeta "github.com/tsenart/vegeta/v12/lib"
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
			addRoute("/one", NewBackendRoute("backend-1"))
			addRoute("/two", NewBackendRoute("backend-2"))
			reloadRoutes()
		})
		AfterEach(func() {
			backend1.Close()
			backend2.Close()
		})

		It("should not significantly increase latency", func() {
			assertPerformantRouter(backend1, backend2)
		})

		Describe("when the routes are being reloaded repeatedly", func() {
			It("should not significantly increase latency", func() {
				stopCh := make(chan struct{})
				defer close(stopCh)
				go func() {
					ticker := time.NewTicker(100 * time.Millisecond)
					defer ticker.Stop()
					select {
					case <-stopCh:
						return
					case <-ticker.C:
						reloadRoutes()
					}
				}()

				assertPerformantRouter(backend1, backend2)
			})
		})

		Describe("one slow backend hit separately", func() {
			It("should not significantly increase latency", func() {
				slowBackend := startTarpitBackend(time.Second)
				defer slowBackend.Close()
				addBackend("backend-slow", slowBackend.URL)
				addRoute("/slow", NewBackendRoute("backend-slow"))
				reloadRoutes()

				attacker := startVegetaLoad(routerURL("/slow"))
				defer attacker.Stop()

				assertPerformantRouter(backend1, backend2)
			})
		})

		Describe("one downed backend hit separately", func() {
			It("should not significantly increase latency", func() {
				addBackend("backend-down", "http://127.0.0.1:3162/")
				addRoute("/down", NewBackendRoute("backend-down"))
				reloadRoutes()

				attacker := startVegetaLoad(routerURL("/down"))
				defer attacker.Stop()

				assertPerformantRouter(backend1, backend2)
			})
		})

		if os.Getenv("RUN_ULIMIT_DEPENDENT_TESTS") != "" {
			Describe("high request throughput", func() {
				It("should not significantly increase latency", func() {
					assertPerformantRouter(backend1, backend2, 3000)
				})
			})
		}
	})

	Describe("many concurrent (slow) connections", func() {
		if os.Getenv("RUN_ULIMIT_DEPENDENT_TESTS") != "" {
			var (
				backend1 *httptest.Server
				backend2 *httptest.Server
			)

			BeforeEach(func() {
				backend1 = startTarpitBackend(time.Second)
				backend2 = startTarpitBackend(time.Second)
				addBackend("backend-1", backend1.URL)
				addBackend("backend-2", backend2.URL)
				addRoute("/one", NewBackendRoute("backend-1"))
				addRoute("/two", NewBackendRoute("backend-2"))
				reloadRoutes()
			})
			AfterEach(func() {
				backend1.Close()
				backend2.Close()
			})

			It("should not significantly increase latency", func() {
				assertPerformantRouter(backend1, backend2, 1000)
			})
		} else {
			PIt("high throughput requires elevated ulimit")
		}
	})
})

func assertPerformantRouter(backend1, backend2 *httptest.Server, optionalRate ...int) {
	var rate = 50
	if len(optionalRate) > 0 {
		rate = optionalRate[0]
	}
	directResultsCh := startVegetaAttack([]string{backend1.URL + "/one", backend2.URL + "/two"}, rate)
	routerResultsCh := startVegetaAttack([]string{routerURL("/one"), routerURL("/two")}, rate)

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

func startVegetaAttack(targetURLs []string, rate int) chan *vegeta.Metrics {
	targets := make([]vegeta.Target, 0, len(targetURLs))
	for _, url := range targetURLs {
		targets = append(targets, vegeta.Target{
			Method: "GET",
			URL:    url,
		})
	}
	targeter := vegeta.NewStaticTargeter(targets...)
	metricsChan := make(chan *vegeta.Metrics, 1)
	go vegetaAttack(targeter, rate, metricsChan)
	return metricsChan
}

func vegetaAttack(targeter vegeta.Targeter, rate int, metricsChan chan *vegeta.Metrics) {
	attacker := vegeta.NewAttacker()

	var metrics vegeta.Metrics
	for res := range attacker.Attack(targeter, vegeta.Pacer(vegeta.ConstantPacer{Freq: rate, Per: 10 * time.Second}), 10*time.Second, "performance-attacker") {
		metrics.Add(res)
	}
	metrics.Close()

	metricsChan <- &metrics
}

func startVegetaLoad(targetURL string) *vegeta.Attacker {
	attacker := vegeta.NewAttacker()
	targetter := vegeta.NewStaticTargeter(vegeta.Target{Method: "GET", URL: targetURL})
	resCh := attacker.Attack(targetter, vegeta.Pacer(vegeta.ConstantPacer{Freq: 50, Per: time.Minute}), time.Minute, "performance-attacker")

	// Consume and discard results. Without this, all the workers will block sending to the channel.
	// TODO: record metrics and use them in tests, rather than discarding them. See
	// https://github.com/tsenart/vegeta#usage-library
	go func() {
		// revive:disable:empty-block
		for range resCh {
			// discard
		}
		// revive:enable:empty-block
	}()
	return attacker
}
