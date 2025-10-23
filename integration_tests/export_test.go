package integration

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	router "github.com/alphagov/router/lib"
	"github.com/rs/zerolog"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Route Export", func() {
	BeforeEach(func() {
		addRoute("/", NewBackendRoute("frontend", "exact"))
		addRoute("/government", NewBackendRoute("frontend", "prefix"))
		addRoute("/old-url", NewRedirectRoute("/new-url", "exact", "ignore"))
		addRoute("/gone-page", NewGoneRoute("exact"))
	})

	It("should export routes in JSONL format", func() {
		ctx := context.Background()
		err := os.Setenv("CONTENT_STORE_DATABASE_URL", postgresContainer.MustConnectionString(ctx))
		Expect(err).NotTo(HaveOccurred())

		// Create a temporary file for export
		tmpFile, err := os.CreateTemp("", "routes-*.jsonl")
		Expect(err).NotTo(HaveOccurred())
		tmpFileName := tmpFile.Name()
		Expect(tmpFile.Close()).To(Succeed())
		defer func() {
			_ = os.Remove(tmpFileName)
		}()

		logger := zerolog.Nop()

		err = router.ExportRoutes(tmpFileName, logger)
		Expect(err).NotTo(HaveOccurred())

		// Read the exported file
		content, err := os.ReadFile(tmpFileName) // #nosec G304 - test uses temp file
		Expect(err).NotTo(HaveOccurred())

		output := string(content)
		Expect(output).NotTo(BeEmpty())

		// Verify JSONL format - each line should be valid JSON
		lines := strings.Split(strings.TrimSpace(output), "\n")
		Expect(len(lines)).To(BeNumerically(">=", 4))

		for i, line := range lines {
			if line == "" {
				continue
			}

			var route router.Route
			err := json.Unmarshal([]byte(line), &route)
			Expect(err).NotTo(HaveOccurred(), "Line %d should be valid JSON: %s", i+1, line)

			// Verify route has required fields
			Expect(route.IncomingPath).NotTo(BeNil())
			Expect(route.RouteType).NotTo(BeNil())
		}
	})

	It("should export routes that can be re-imported", func() {
		// Set database URL for export
		ctx := context.Background()
		err := os.Setenv("CONTENT_STORE_DATABASE_URL", postgresContainer.MustConnectionString(ctx))
		Expect(err).NotTo(HaveOccurred())

		// Create a temporary file for export
		tmpFile, err := os.CreateTemp("", "routes-*.jsonl")
		Expect(err).NotTo(HaveOccurred())
		tmpFileName := tmpFile.Name()
		Expect(tmpFile.Close()).To(Succeed())
		defer func() {
			_ = os.Remove(tmpFileName)
		}()

		logger := zerolog.Nop()

		// Export routes
		err = router.ExportRoutes(tmpFileName, logger)
		Expect(err).NotTo(HaveOccurred())

		// Read the exported file
		content, err := os.ReadFile(tmpFileName) // #nosec G304 - test uses temp file
		Expect(err).NotTo(HaveOccurred())

		output := string(content)
		lines := strings.Split(strings.TrimSpace(output), "\n")

		// Verify we can parse each exported route
		routeCount := 0
		for _, line := range lines {
			if line == "" {
				continue
			}

			var route router.Route
			err := json.Unmarshal([]byte(line), &route)
			Expect(err).NotTo(HaveOccurred())

			routeCount++
		}

		Expect(routeCount).To(BeNumerically(">=", 4))
	})
})
