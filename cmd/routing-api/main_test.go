package main_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
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
		session := RoutingApi("-config=../../example_config/bad_uaa_verification_key.yml", "-ip='127.0.0.1'", "-systemDomain='domain'", "-consulCluster="+consulRunner.ConsulCluster(), etcdUrl)
		Eventually(session).Should(Exit(1))
		Eventually(session).Should(Say("Public uaa token must be PEM encoded"))
	})

	It("exits 1 if the consulCluster is not provided", func() {
		session := RoutingApi("-config=../../example_config/example.yml", "-ip='127.0.0.1'", "-systemDomain='domain'", etcdUrl)
		Eventually(session).Should(Exit(1))
		Eventually(session).Should(Say("No consul cluster provided"))
	})

	Context("when initialized correctly and etcd is running", func() {
		It("unregisters form etcd when the process exits", func() {
			session := RoutingApi("-config=../../example_config/example.yml", "-ip='127.0.0.1'", "-systemDomain='domain'", "-consulCluster="+consulRunner.ConsulCluster(), etcdUrl)

			getRoutes := func() string {
				routesPath := fmt.Sprintf("%s/v2/keys/routes", etcdUrl)
				resp, err := http.Get(routesPath)
				Expect(err).ToNot(HaveOccurred())

				body, err := ioutil.ReadAll(resp.Body)
				Expect(err).ToNot(HaveOccurred())
				return string(body)
			}
			Eventually(getRoutes).Should(ContainSubstring("routing-api"))

			session.Terminate()

			Eventually(getRoutes).ShouldNot(ContainSubstring("routing-api"))
			Eventually(session.ExitCode()).Should(Equal(0))
		})

		It("exits 1 if etcd returns an error as we unregister ourself during a deployment roll", func() {
			session := RoutingApi("-config=../../example_config/example.yml", "-ip='127.0.0.1'", "-systemDomain='domain'", "-consulCluster="+consulRunner.ConsulCluster(), etcdUrl)

			etcdAdapter.Disconnect()
			etcdRunner.Stop()

			session.Terminate()

			Eventually(session).Should(Exit(1))
		})
	})
})

func RoutingApi(args ...string) *Session {
	path, err := Build("github.com/cloudfoundry-incubator/routing-api/cmd/routing-api")
	Expect(err).NotTo(HaveOccurred())

	session, err := Start(exec.Command(path, args...), GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())

	return session
}
