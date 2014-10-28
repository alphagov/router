package integration

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestEverything(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration test suite")
}

var _ = BeforeSuite(func() {
	var err error
	err = setupTempLogfile()
	if err != nil {
		Fail(err.Error())
	}
	err = startRouter(3169, 3168)
	if err != nil {
		Fail(err.Error())
	}
})

var _ = BeforeEach(func() {
	resetTempLogfile()
})

var _ = AfterSuite(func() {
	stopRouter(3169)
	cleanupTempLogfile()
})
