package integration

import (
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
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

	backendEnvVars := []string{}
	for id, host := range backends {
		envVar := "BACKEND_URL_" + id + "=http://" + host
		backendEnvVars = append(backendEnvVars, envVar)
	}

	err = startRouter(routerPort, apiPort, backendEnvVars)
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
