package router

import (
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
)

var _ = Describe("Backends", func() {
	var (
		logger = zerolog.New(os.Stdout)
	)

	Context("When calling loadBackendsFromEnv", func() {
		It("should load backends from environment variables", func() {
			os.Setenv("BACKEND_URL_testBackend", "http://example.com")
			defer os.Unsetenv("BACKEND_URL_testBackend")

			backends := loadBackendsFromEnv(1*time.Second, 20*time.Second, logger)

			Expect(backends).To(HaveKey("testBackend"))
			Expect(backends["testBackend"]).ToNot(BeNil())
		})

		It("should skip backends with empty URLs", func() {
			os.Setenv("BACKEND_URL_emptyBackend", "")
			defer os.Unsetenv("BACKEND_URL_emptyBackend")

			backends := loadBackendsFromEnv(1*time.Second, 20*time.Second, logger)

			Expect(backends).ToNot(HaveKey("emptyBackend"))
		})

		It("should skip backends with invalid URLs", func() {
			os.Setenv("BACKEND_URL_invalidBackend", "://invalid-url")
			defer os.Unsetenv("BACKEND_URL_invalidBackend")

			backends := loadBackendsFromEnv(1*time.Second, 20*time.Second, logger)

			Expect(backends).ToNot(HaveKey("invalidBackend"))
		})
	})
})
