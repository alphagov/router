package integration

import (
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	vegeta "github.com/tsenart/vegeta/v12/lib"
)

const routerLatencyThreshold = 20 * time.Millisecond

var _ = Describe("Performance", func() {

	Context("with two healthy backends", func() {
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
			reloadRoutes(apiPort)
		})
		AfterEach(func() {
			backend1.Close()
			backend2.Close()
		})

		It("Router should not cause errors or much latency", func() {
			assertPerformantRouter(backend1, backend2, 100)
		})

		Describe("when the routes are being reloaded repeatedly", func() {
			It("Router should not cause errors or much latency", func() {
				stopCh := make(chan struct{})
				defer close(stopCh)
				go func() {
					ticker := time.NewTicker(100 * time.Millisecond)
					defer ticker.Stop()
					select {
					case <-stopCh:
						return
					case <-ticker.C:
						reloadRoutes(apiPort)
					}
				}()

				assertPerformantRouter(backend1, backend2, 100)
			})
		})

		Describe("with one slow backend hit separately", func() {
			It("Router should not cause errors or much latency", func() {
				slowBackend := startTarpitBackend(time.Second)
				defer slowBackend.Close()
				addBackend("backend-slow", slowBackend.URL)
				addRoute("/slow", NewBackendRoute("backend-slow"))
				reloadRoutes(apiPort)

				_, gen := generateLoad([]string{routerURL(routerPort, "/slow")}, 50)
				defer gen.Stop()

				assertPerformantRouter(backend1, backend2, 50)
			})
		})

		Describe("with one downed backend hit separately", func() {
			It("Router should not cause errors or much latency", func() {
				addBackend("backend-down", "http://127.0.0.1:3162/")
				addRoute("/down", NewBackendRoute("backend-down"))
				reloadRoutes(apiPort)

				_, gen := generateLoad([]string{routerURL(routerPort, "/down")}, 50)
				defer gen.Stop()

				assertPerformantRouter(backend1, backend2, 50)
			})
		})

		Describe("with high request throughput", func() {
			It("Router should not cause errors or much latency", func() {
				assertPerformantRouter(backend1, backend2, 500)
			})
		})
	})

	Describe("many concurrent slow connections", func() {
		var backend1 *httptest.Server
		var backend2 *httptest.Server

		BeforeEach(func() {
			backend1 = startTarpitBackend(time.Second)
			backend2 = startTarpitBackend(time.Second)
			addBackend("backend-1", backend1.URL)
			addBackend("backend-2", backend2.URL)
			addRoute("/one", NewBackendRoute("backend-1"))
			addRoute("/two", NewBackendRoute("backend-2"))
			reloadRoutes(apiPort)
		})
		AfterEach(func() {
			backend1.Close()
			backend2.Close()
		})

		It("Router should not cause errors or much latency", func() {
			assertPerformantRouter(backend1, backend2, 500)
		})
	})
})

func assertPerformantRouter(backend1, backend2 *httptest.Server, rps int) {
	directResultsCh, _ := generateLoad([]string{backend1.URL + "/one", backend2.URL + "/two"}, rps)
	routerResultsCh, _ := generateLoad([]string{routerURL(routerPort, "/one"), routerURL(routerPort, "/two")}, rps)

	directResults := <-directResultsCh
	routerResults := <-routerResultsCh

	Expect(routerResults.Requests).To(Equal(directResults.Requests))
	Expect(routerResults.Success).To(BeNumerically("~", 1.0))
	Expect(directResults.Success).To(BeNumerically("~", 1.0))

	Expect(routerResults.Latencies.Mean).To(BeNumerically("~", directResults.Latencies.Mean, routerLatencyThreshold))
	Expect(routerResults.Latencies.P95).To(BeNumerically("~", directResults.Latencies.P95, routerLatencyThreshold))
	Expect(routerResults.Latencies.P99).To(BeNumerically("~", directResults.Latencies.P99, routerLatencyThreshold*2))
	Expect(routerResults.Latencies.Max).To(BeNumerically("~", directResults.Latencies.Max, routerLatencyThreshold*2))
}

func generateLoad(targetURLs []string, rps int) (chan *vegeta.Metrics, *vegeta.Attacker) {
	targets := make([]vegeta.Target, 0, len(targetURLs))
	for _, url := range targetURLs {
		targets = append(targets, vegeta.Target{
			Method: "GET",
			URL:    url,
		})
	}
	targeter := vegeta.NewStaticTargeter(targets...)
	metrics := make(chan *vegeta.Metrics, 1)
	veg := vegeta.NewAttacker()
	go vegetaAttack(veg, targeter, rps, metrics)
	return metrics, veg
}

func vegetaAttack(veg *vegeta.Attacker, targets vegeta.Targeter, rps int, metrics chan *vegeta.Metrics) {
	pace := vegeta.Pacer(vegeta.ConstantPacer{Freq: rps, Per: time.Second})

	var m vegeta.Metrics
	for res := range veg.Attack(targets, pace, 10*time.Second, "load") {
		m.Add(res)
	}
	m.Close()

	metrics <- &m
}
