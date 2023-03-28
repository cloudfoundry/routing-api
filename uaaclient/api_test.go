package uaaclient_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/lager/v3/lagertest"
	"code.cloudfoundry.org/routing-api/uaaclient"
)

var _ = Describe("NewAPI", func() {
	var (
		config uaaclient.Config
		logger lager.Logger
	)

	BeforeEach(func() {
		config = uaaclient.Config{
			SkipSSLValidation: true,
			ClientName:        "client-name",
			ClientSecret:      "client-secret",
			TokenEndpoint:     "10.10.10.10",
			Port:              8443,
		}
		logger = lagertest.NewTestLogger("test")
	})

	Context("when protocol is specified", func() {
		BeforeEach(func() {
			config.Protocol = "http"
		})
		It("returns an api obj using the right protocol", func() {
			api, err := uaaclient.NewAPI(config, logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(api.TargetURL.String()).To(Equal("http://10.10.10.10:8443"))
		})
	})
	Context("when protocol is not specified", func() {
		It("returns an api obj using the right protocol", func() {
			api, err := uaaclient.NewAPI(config, logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(api.TargetURL.String()).To(Equal("https://10.10.10.10:8443"))
		})
	})
})
