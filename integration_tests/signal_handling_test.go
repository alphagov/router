package integration

import (
	"fmt"
	"log"
	"net/http"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = XDescribe("Signal Handling", func() {
	const signalHandlingRouterPort = routerPort + 2
	const signalHandlingApiPort = apiPort + 2

	BeforeEach(func() {
		StartupRouterForIntegrationTests(signalHandlingRouterPort, signalHandlingApiPort)
	})

	AfterEach(func() {
		err := ensureRouterTerminated(signalHandlingRouterPort)
		if err != nil {
			log.Fatalf("Couldn't terminate router successfully, can't continue tests, error was: %s", err)
		}
	})

	for _, signal := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM} {
		Context(fmt.Sprintf("When receiving a %s", signal), func() {
			It("Exits successfully", func() {
				routerCmd, err := sendSignalToRouter(signalHandlingRouterPort, signal)
				Expect(err).ToNot(HaveOccurred())
				state, err := routerCmd.Process.Wait()
				Expect(state.Success()).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())
			})

			It("Gracefully allows connections to complete", func() {
				// Use a backend request that takes 1 seconds to give us time to send the signal to router
				slowBackend := startTarpitBackend(backends["signals-1"], time.Second)
				defer slowBackend.Close()
				addRoute("/signals", NewBackendRoute("signals-1"))
				reloadRoutes(signalHandlingApiPort)

				responseChannel := make(chan *AsyncResponse, 1)
				go doRequestAsync(
					newRequest(http.MethodGet, routerURL(signalHandlingRouterPort, "/signals")),
					responseChannel,
				)

				// Sleep for just a little time to give the request chance to start
				time.Sleep(time.Millisecond * 100)

				_, err := sendSignalToRouter(signalHandlingRouterPort, signal)
				Expect(err).ToNot(HaveOccurred())

				response := <-responseChannel

				Expect(response.err).ToNot(HaveOccurred())
				Expect(response.response.StatusCode).To(Equal(200))
			})
		})
	}
})
