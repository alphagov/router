package integration

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEverything(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration test suite")
}

var _ = BeforeSuite(func() {
	err := setupTempLogfile()
	if err != nil {
		Fail(err.Error())
	}
	err = startRouter(routerPort, apiPort, nil)
	if err != nil {
		Fail(err.Error())
	}
	err = initRouteHelper()
	if err != nil {
		Fail(err.Error())
	}
})

var _ = BeforeEach(func() {
	resetTempLogfile()
})

var _ = AfterSuite(func() {
	stopRouter(routerPort)
	cleanupTempLogfile()
})
