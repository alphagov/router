package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/alphagov/router/triemux"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pashagolub/pgxmock/v4"
)

var _ = Describe("loadRoutesFromCS", func() {
	var (
		mockPool pgxmock.PgxPoolIface
		mux      *triemux.Mux
		backends map[string]http.Handler
	)

	BeforeEach(func() {
		var err error
		mockPool, err = pgxmock.NewPool()
		Expect(err).NotTo(HaveOccurred())

		mux = triemux.NewMux()
		backends = map[string]http.Handler{
			"backend1": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("backend1")) //nolint:errcheck
			}),
			"backend2": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("backend2")) //nolint:errcheck
			}),
			"government-frontend": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("government-frontend")) //nolint:errcheck
			}),
		}
	})

	AfterEach(func() {
		mockPool.Close()
	})

	Context("when content store has backend routes", func() {
		BeforeEach(func() {
			rows := pgxmock.NewRows([]string{"backend", "path", "match_type", "destination", "segments_mode", "schema_name", "details"}).
				AddRow(stringPtr("backend1"), stringPtr("/path1"), stringPtr("exact"), nil, nil, stringPtr("guidance"), stringPtr("")).
				AddRow(stringPtr("backend2"), stringPtr("/path2"), stringPtr("prefix"), nil, nil, stringPtr("guidance"), stringPtr(""))

			mockPool.ExpectQuery("WITH").WillReturnRows(rows)

			err := loadRoutesFromCS(mockPool, mux, backends)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should load backend exact routes correctly", func() {
			req, _ := http.NewRequest(http.MethodGet, "/path1", nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(rr.Body.String()).To(Equal("backend1"))
		})

		It("should load backend prefix routes correctly", func() {
			req, _ := http.NewRequest(http.MethodGet, "/path2/foo/bar", nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(rr.Body.String()).To(Equal("backend2"))
		})
	})

	Context("when content store has gone routes", func() {
		BeforeEach(func() {
			rows := pgxmock.NewRows([]string{"backend", "path", "match_type", "destination", "segments_mode", "schema_name", "details"}).
				AddRow(nil, stringPtr("/government-frontend-gone"), stringPtr("exact"), nil, nil, stringPtr("gone"), stringPtr("{\"explanation\": \"this is gone\", \"alternative_path\": null}")).
				AddRow(stringPtr("backend2"), stringPtr("/backend-gone"), stringPtr("exact"), nil, nil, stringPtr("gone"), stringPtr("{\"explanation\": \"this is gone\", \"alternative_path\": null}")).
				AddRow(stringPtr("backend1"), stringPtr("/guidance"), stringPtr("exact"), nil, nil, stringPtr("guidance"), stringPtr("")).
				AddRow(nil, stringPtr("/gone-empty-attributes"), stringPtr("exact"), nil, nil, stringPtr("gone"), stringPtr("{\"explanation\": null, \"alternative_path\": null}")).
				AddRow(nil, stringPtr("/gone-empty-details"), stringPtr("exact"), nil, nil, stringPtr("gone"), stringPtr("{}")).
				AddRow(nil, stringPtr("/gone-nil-details"), stringPtr("exact"), nil, nil, stringPtr("gone"), nil)

			mockPool.ExpectQuery("WITH").WillReturnRows(rows)

			err := loadRoutesFromCS(mockPool, mux, backends)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should load gone route with description with empty fields", func() {
			req, _ := http.NewRequest(http.MethodGet, "/gone-empty-attributes", nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusGone))
		})

		It("should load gone route with empty description", func() {
			req, _ := http.NewRequest(http.MethodGet, "/gone-empty-details", nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusGone))
		})

		It("should load gone route with nil description", func() {
			req, _ := http.NewRequest(http.MethodGet, "/gone-nil-details", nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusGone))
		})

		It("should load backend route with description and backend", func() {
			req, _ := http.NewRequest(http.MethodGet, "/backend-gone", nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(rr.Body.String()).To(Equal("backend2"))
		})

		It("should load government-frontend backend route with description and without backend", func() {
			req, _ := http.NewRequest(http.MethodGet, "/government-frontend-gone", nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(rr.Body.String()).To(Equal("government-frontend"))
		})
	})

	Context("when content store has redirect routes", func() {
		BeforeEach(func() {
			rows := pgxmock.NewRows([]string{"backend", "path", "match_type", "destination", "segments_mode", "schema_name", "details"}).
				AddRow(nil, stringPtr("/redirect-exact"), stringPtr("exact"), stringPtr("/redirected-exact"), nil, stringPtr("redirect"), nil).
				AddRow(nil, stringPtr("/redirect-prefix"), stringPtr("prefix"), stringPtr("/redirected-prefix"), nil, stringPtr("redirect"), nil).
				AddRow(nil, stringPtr("/redirect-exact-ignore"), stringPtr("exact"), stringPtr("/redirected-exact-ignore"), stringPtr("ignore"), stringPtr("redirect"), nil).
				AddRow(nil, stringPtr("/redirect-prefix-ignore"), stringPtr("prefix"), stringPtr("/redirected-prefix-ignore"), stringPtr("ignore"), stringPtr("redirect"), nil).
				AddRow(nil, stringPtr("/redirect-exact-preserve"), stringPtr("exact"), stringPtr("/redirected-exact-preserve"), stringPtr("preserve"), stringPtr("redirect"), nil).
				AddRow(nil, stringPtr("/redirect-prefix-preserve"), stringPtr("prefix"), stringPtr("/redirected-prefix-preserve"), stringPtr("preserve"), stringPtr("redirect"), nil)
			mockPool.ExpectQuery("WITH").WillReturnRows(rows)

			err := loadRoutesFromCS(mockPool, mux, backends)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should load exact redirect route", func() {
			req, _ := http.NewRequest(http.MethodGet, "/redirect-exact", nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusMovedPermanently))
			Expect(rr.Header().Get("Location")).To(Equal("/redirected-exact"))
		})

		It("should load prefix redirect route", func() {
			req, _ := http.NewRequest(http.MethodGet, "/redirect-prefix/foo/bar", nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusMovedPermanently))
			Expect(rr.Header().Get("Location")).To(Equal("/redirected-prefix/foo/bar"))
		})

		It("should load exact redirect route that ignores suffix segments", func() {
			req, _ := http.NewRequest(http.MethodGet, "/redirect-exact-ignore", nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusMovedPermanently))
			Expect(rr.Header().Get("Location")).To(Equal("/redirected-exact-ignore"))
		})

		It("should load prefix redirect route that ignores suffix segments", func() {
			req, _ := http.NewRequest(http.MethodGet, "/redirect-prefix-ignore/foo/bar", nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusMovedPermanently))
			Expect(rr.Header().Get("Location")).To(Equal("/redirected-prefix-ignore"))
		})

		It("should load exact redirect route that preserves suffix segments", func() {
			req, _ := http.NewRequest(http.MethodGet, "/redirect-exact-preserve", nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusMovedPermanently))
			Expect(rr.Header().Get("Location")).To(Equal("/redirected-exact-preserve"))
		})

		It("should load prefix redirect route that preserves suffix segments", func() {
			req, _ := http.NewRequest(http.MethodGet, "/redirect-prefix-preserve/foo/bar", nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusMovedPermanently))
			Expect(rr.Header().Get("Location")).To(Equal("/redirected-prefix-preserve/foo/bar"))
		})
	})
})

var _ = Describe("Router", func() {
	Describe("reloadCsRoutes", func() {
		var (
			mockPool pgxmock.PgxPoolIface
			router   *Router
		)

		BeforeEach(func() {
			var err error
			mockPool, err = pgxmock.NewPool()
			Expect(err).NotTo(HaveOccurred())

			router = &Router{
				lock: sync.RWMutex{},
				backends: map[string]http.Handler{
					"backend1": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						http.Redirect(w, r, "http://example.com", http.StatusFound)
					}),
					"backend2": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						http.Redirect(w, r, "http://example.com", http.StatusFound)
					}),
				},
			}
		})

		AfterEach(func() {
			mockPool.Close()
		})

		It("should reload routes from content store successfully", func() {
			rows := pgxmock.NewRows([]string{"backend", "path", "match_type", "destination", "segments_mode", "schema_name", "details"}).
				AddRow(stringPtr("backend1"), stringPtr("/path1"), stringPtr("exact"), nil, nil, stringPtr("guidance"), stringPtr("")).
				AddRow(stringPtr("backend2"), stringPtr("/path2"), stringPtr("prefix"), nil, nil, stringPtr("guidance"), stringPtr(""))

			mockPool.ExpectQuery("WITH").WillReturnRows(rows)

			router.reloadCsRoutes(mockPool)

			Expect(router.csMux.RouteCount()).To(Equal(2))
		})

		It("should handle panic and log error", func() {
			defer GinkgoRecover()

			mockPool.ExpectQuery("WITH").WillReturnError(fmt.Errorf("some error"))

			Expect(func() { router.reloadCsRoutes(mockPool) }).NotTo(Panic())
		})
	})
})
