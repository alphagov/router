package router

import (
	"fmt"
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
			if err := os.Setenv("BACKEND_URL_testBackend", "http://example.com"); err != nil {
				Fail(fmt.Sprintf("Couldn't set up test, failed to Setenv, %v", err))
			}

			defer func() {
				if err := os.Unsetenv("BACKEND_URL_testBackend"); err != nil {
					fmt.Println("Failed to unset env", err)
				}
			}()

			backends := loadBackendsFromEnv(1*time.Second, 20*time.Second, logger)

			Expect(backends).To(HaveKey("testBackend"))
			Expect(backends["testBackend"]).ToNot(BeNil())
		})

		It("should skip backends with empty URLs", func() {
			if err := os.Setenv("BACKEND_URL_emptyBackend", ""); err != nil {
				Fail(fmt.Sprintf("Couldn't set up test, failed to Setenv, %v", err))
			}
			defer func() {
				if err := os.Unsetenv("BACKEND_URL_emptyBackend"); err != nil {
					fmt.Println("Failed to unset env", err)
				}
			}()

			backends := loadBackendsFromEnv(1*time.Second, 20*time.Second, logger)

			Expect(backends).ToNot(HaveKey("emptyBackend"))
		})

		It("should skip backends with invalid URLs", func() {
			if err := os.Setenv("BACKEND_URL_invalidBackend", "://invalid-url"); err != nil {
				Fail(fmt.Sprintf("Couldn't set up test, failed to Setenv, %v", err))
			}
			defer func() {
				if err := os.Unsetenv("BACKEND_URL_invalidBackend"); err != nil {
					fmt.Println("Failed to unset env", err)
				}
			}()

			backends := loadBackendsFromEnv(1*time.Second, 20*time.Second, logger)

			Expect(backends).ToNot(HaveKey("invalidBackend"))
		})
	})
})
