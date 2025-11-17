package integration

import (
	"context"
	"net/http/httptest"
	"time"

	"github.com/jackc/pgx/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("database unavailable", Serial, func() {
	var (
		backend1            *httptest.Server
		containerWasStopped bool
		skipRouteCleanup    bool
	)

	BeforeEach(func() {
		backend1 = startSimpleBackend("backend 1", backends["backend-1"])
		containerWasStopped = false
		skipRouteCleanup = false
	})

	AfterEach(func() {
		backend1.Close()

		// Ensure postgres container is running after the test
		// This is critical for test isolation
		ctx := context.Background()
		if containerWasStopped || !postgresContainer.IsRunning() {
			err := postgresContainer.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
			// Wait for postgres to be ready
			time.Sleep(2 * time.Second)

			// Reconnect the pgConn since the old connection is dead
			databaseURL := postgresContainer.MustConnectionString(ctx)
			var err2 error
			pgConn, err2 = pgx.Connect(ctx, databaseURL)
			Expect(err2).NotTo(HaveOccurred())
		}

		// Skip route cleanup if we stopped the container
		// The shared AfterEach will fail with a dead connection
		if !skipRouteCleanup {
			clearRoutes()
		}
	})

	Context("when postgres becomes unavailable after initial route load", func() {
		var ctx context.Context

		BeforeEach(func() {
			ctx = context.Background()

			// Load initial routes into the database
			addRoute("/test-backend", NewBackendRoute("backend-1"))
			addRoute("/test-redirect", NewRedirectRoute("/redirected", "exact"))
			addRoute("/test-prefix", NewBackendRoute("backend-1", "prefix"))

			// Reload routes to ensure router has loaded them
			reloadRoutes(apiPort)
		})

		It("should continue serving existing routes when database is stopped", func() {
			// Verify routes work before stopping the database (baseline)
			resp := routerRequest(routerPort, "/test-backend")
			Expect(resp.StatusCode).To(Equal(200))
			Expect(readBody(resp)).To(Equal("backend 1"))

			resp = routerRequest(routerPort, "/test-redirect")
			Expect(resp.StatusCode).To(Equal(301))
			Expect(resp.Header.Get("Location")).To(Equal("/redirected"))

			resp = routerRequest(routerPort, "/test-prefix/foo")
			Expect(resp.StatusCode).To(Equal(200))
			Expect(readBody(resp)).To(Equal("backend 1"))

			// Stop the PostgreSQL container to simulate database failure
			stopTimeout := 5 * time.Second
			err := postgresContainer.Stop(ctx, &stopTimeout)
			Expect(err).NotTo(HaveOccurred())
			containerWasStopped = true
			skipRouteCleanup = true

			// Wait briefly to ensure the database is actually down
			time.Sleep(100 * time.Millisecond)

			// Verify routes still work with database down
			resp = routerRequest(routerPort, "/test-backend")
			Expect(resp.StatusCode).To(Equal(200))
			Expect(readBody(resp)).To(Equal("backend 1"))

			resp = routerRequest(routerPort, "/test-redirect")
			Expect(resp.StatusCode).To(Equal(301))
			Expect(resp.Header.Get("Location")).To(Equal("/redirected"))

			resp = routerRequest(routerPort, "/test-prefix/foo")
			Expect(resp.StatusCode).To(Equal(200))
			Expect(readBody(resp)).To(Equal("backend 1"))

			// Verify multiple requests work (no transient failures)
			for i := 0; i < 5; i++ {
				resp = routerRequest(routerPort, "/test-backend")
				Expect(resp.StatusCode).To(Equal(200))
			}
		})

		It("should preserve routes after failed reload attempts", func() {
			// Stop the PostgreSQL container
			stopTimeout := 5 * time.Second
			err := postgresContainer.Stop(ctx, &stopTimeout)
			Expect(err).NotTo(HaveOccurred())
			containerWasStopped = true
			skipRouteCleanup = true

			// Wait for database to be down
			time.Sleep(100 * time.Millisecond)

			// Attempt to reload routes (this should fail silently)
			// The API should still return 202 (reload queued)
			resp := doRequest(newRequest("POST", routerURL(apiPort, "/reload")))
			Expect(resp.StatusCode).To(Equal(202))

			// Wait for reload attempt to complete (and fail)
			time.Sleep(200 * time.Millisecond)

			// Verify routes still work (old routes preserved)
			resp = routerRequest(routerPort, "/test-backend")
			Expect(resp.StatusCode).To(Equal(200))
			Expect(readBody(resp)).To(Equal("backend 1"))

			resp = routerRequest(routerPort, "/test-redirect")
			Expect(resp.StatusCode).To(Equal(301))
		})

		It("should recover and load new routes when database comes back", func() {
			// Stop the PostgreSQL container
			stopTimeout := 5 * time.Second
			err := postgresContainer.Stop(ctx, &stopTimeout)
			Expect(err).NotTo(HaveOccurred())
			containerWasStopped = true
			time.Sleep(100 * time.Millisecond)

			// Verify old routes still work
			resp := routerRequest(routerPort, "/test-backend")
			Expect(resp.StatusCode).To(Equal(200))

			// Start the PostgreSQL container
			err = postgresContainer.Start(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Wait for postgres to be ready and reconnect
			time.Sleep(2 * time.Second)
			databaseURL := postgresContainer.MustConnectionString(ctx)
			pgConn, err = pgx.Connect(ctx, databaseURL)
			Expect(err).NotTo(HaveOccurred())

			// Verify connection works
			Eventually(func() error {
				_, err := pgConn.Exec(ctx, "SELECT 1")
				return err
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

			// Add a new route to the database
			addRoute("/new-route", NewBackendRoute("backend-1"))

			// Reload routes
			reloadRoutes(apiPort)

			// Verify new route works
			Eventually(func() int {
				return routerRequest(routerPort, "/new-route").StatusCode
			}, 3*time.Second, 100*time.Millisecond).Should(Equal(200))

			// Verify old routes still work
			resp = routerRequest(routerPort, "/test-backend")
			Expect(resp.StatusCode).To(Equal(200))
			Expect(readBody(resp)).To(Equal("backend 1"))
		})

		It("should handle database outage without panicking or crashing", func() {
			// Stop the database
			stopTimeout := 5 * time.Second
			err := postgresContainer.Stop(ctx, &stopTimeout)
			Expect(err).NotTo(HaveOccurred())
			containerWasStopped = true
			skipRouteCleanup = true
			time.Sleep(100 * time.Millisecond)

			// Make many requests to stress test the router
			for i := 0; i < 20; i++ {
				resp := routerRequest(routerPort, "/test-backend")
				Expect(resp.StatusCode).To(Equal(200))

				// Verify API endpoints still work
				healthResp := doRequest(newRequest("GET", routerURL(apiPort, "/healthcheck")))
				Expect(healthResp.StatusCode).To(Equal(200))
			}

			// The router should still be responsive
			resp := routerRequest(routerPort, "/test-redirect")
			Expect(resp.StatusCode).To(Equal(301))
		})
	})
})
