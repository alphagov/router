package router

import (
	"testing"

	"net/http"
	"net/http/httptest"

	"github.com/pashagolub/pgxmock/v4"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCSMux(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CSMux Suite")
}

var _ = Describe("ContentStoreMux", func() {
	DescribeTable("ContentStoreMux", func(path, backend, schema, destination, segments_mode string, expectedStatus int, expectedBody string) {
		backends := map[string]http.Handler{
			"backend1": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Hello from backend1"))
			}),
			"backend2": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Hello from backend2"))
			}),
		}

		mock, err := pgxmock.NewPool()
		if err != nil {
			Fail("Failed to create mock pool: " + err.Error())
		}
		defer mock.Close()

		rows := mock.NewRows([]string{"path", "type", "rendering_app", "destination", "segments_mode", "schema_name"}).
			AddRow(path, "exact", backend, destination, nil, schema)

		mock.ExpectQuery("WITH unnested_routes AS").WithArgs(path).WillReturnRows(rows)

		// Create a ContentStoreMux instance
		mux := NewCSMux(mock)

		// Create a test request
		req, err := http.NewRequest(http.MethodGet, path, nil)
		Expect(err).NotTo(HaveOccurred())

		// Create a response recorder to capture the responsegit st
		rr := httptest.NewRecorder()

		// Call the ServeHTTP method
		mux.ServeHTTP(rr, req, &backends)

		// Assert the response status code
		Expect(rr.Code).To(Equal(expectedStatus))

		// Assert the response body
		Expect(rr.Body.String()).To(Equal(expectedBody))
	},
		Entry(nil, "/test", "backend1", "guide", nil, nil, http.StatusOK, "Hello from backend1"),
		Entry(nil, "/anothet", "backend2", "guide", nil, nil, http.StatusOK, "Hello from backend2"),
		Entry(nil, "/redirect", nil, "redirect", "/redirected", "ignore", http.StatusMovedPermanently, "<a href=\"/redirected\">Moved Permanently</a>.\n\n"),
		Entry(nil, "/redirect/a/b", nil, "redirect", "/redirected/a/b", "preserve", http.StatusMovedPermanently, "<a href=\"/redirected/a/b\">Moved Permanently</a>.\n\n"),
		Entry(nil, "/gone-test", nil, "gone", nil, nil, http.StatusGone, "410 Gone\n"),
	)
})
