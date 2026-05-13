package integration

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("executing with help flag", func() {
	var cmd *exec.Cmd

	BeforeEach(func() {
		backgroundContext := context.Background()
		coverageDir, err := getCoverageDir()
		Expect(err).NotTo(HaveOccurred())

		cmd = exec.CommandContext(backgroundContext, routerBinary(), "--help") //gosec:disable G204 //gosec:disable G702-- We intentionally want to exec a sub process with a var
		cmd.Env = append(cmd.Env, fmt.Sprintf("GOCOVERDIR=%s", coverageDir))
	})

	It("should exit with error code 64", func() {
		err := cmd.Run()
		Expect(err).To(HaveOccurred())
		exitError, ok := errors.AsType[*exec.ExitError](err)
		Expect(ok).To(BeTrueBecause("The reason the command failed was that the router binary exited non-zero"))
		Expect(exitError.ExitCode()).To(Equal(64))
	})

	It("Should print out usage information", func() {
		output, _ := cmd.CombinedOutput()
		Expect(output).To(ContainSubstring("Usage: "))
	})
})
