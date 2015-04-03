package main_test

import (
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Main", func() {
	It("exits 1 if no config file is provided", func() {
		session := RoutingApi()
		Eventually(session).Should(Exit(1))
		Eventually(session).Should(Say("No configuration file provided"))
	})

	It("exits 1 if no ip address is provided", func() {
		session := RoutingApi("-config=../../example_config/example.yml")
		Eventually(session).Should(Exit(1))
		Eventually(session).Should(Say("No ip address provided"))
	})

	It("exits 1 if no system domain is provided", func() {
		session := RoutingApi("-config=../../example_config/example.yml", "-ip='1.1.1.1'")
		Eventually(session).Should(Exit(1))
		Eventually(session).Should(Say("No system domain provided"))
	})

	It("exits 1 if the uaa_verification_key is not a valid PEM format", func() {
		session := RoutingApi("-config=../../example_config/bad_uaa_verification_key.yml", "-ip='1.1.1.1'", "-systemDomain='some-system-domain'")
		Eventually(session).Should(Exit(1))
		Eventually(session).Should(Say("Public uaa token must be PEM encoded"))
	})
})

func RoutingApi(args ...string) *Session {
	path, err := Build("github.com/cloudfoundry-incubator/routing-api/cmd/routing-api")
	Expect(err).NotTo(HaveOccurred())

	session, err := Start(exec.Command(path, args...), GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())

	return session
}
