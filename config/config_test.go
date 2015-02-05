package config_test

import (
	"github.com/cloudfoundry-incubator/routing-api/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Describe("NewConfigFromFile", func() {
		Context("when the file exists", func() {
			It("returns a valid Config struct", func() {
				cfg_file := "../example_config/example.yml"
				_, err := config.NewConfigFromFile(cfg_file)

				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the file does not exists", func() {
			It("returns an error", func() {
				cfg_file := "notexist"
				_, err := config.NewConfigFromFile(cfg_file)

				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Initialize", func() {
		var (
			cfg *config.Config
		)

		BeforeEach(func() {
			cfg = &config.Config{}
		})

		Context("With a proper yml file", func() {
			test_config := `uaa_verification_key: "public_key"`

			It("sets the UaaPublicKey", func() {
				err := cfg.Initialize([]byte(test_config))
				Expect(err).ToNot(HaveOccurred())

				Expect(cfg.UAAPublicKey).To(Equal("public_key"))
			})
		})

		Context("when there are errors in the yml file", func() {
			test_config := `
uaa:
`
			It("errors if no UaaPublicKey is found", func() {
				err := cfg.Initialize([]byte(test_config))
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
