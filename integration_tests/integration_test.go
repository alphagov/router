package integration

import (
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestEverything(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration test suite")
}

var _ = BeforeSuite(func() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	var err error
	err = setupTempLogfile()
	if err != nil {
		Fail(err.Error())
	}
	err = startRouter(3168, 3168)
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
	stopRouter(5432)
	cleanupTempLogfile()
})
