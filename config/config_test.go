package config_test

import (
	"errors"
	"fmt"
	"time"

	"code.cloudfoundry.org/locket"
	"code.cloudfoundry.org/routing-api/config"
	"code.cloudfoundry.org/routing-api/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Describe("NewConfigFromFile", func() {
		Context("when auth is enabled", func() {
			Context("when the file exists", func() {
				It("returns a valid Config struct", func() {
					cfg_file := "../example_config/example.yml"
					cfg, err := config.NewConfigFromFile(cfg_file, false)

					Expect(err).NotTo(HaveOccurred())
					Expect(cfg.API.ListenPort).To(Equal(3000))
					Expect(cfg.AdminPort).To(Equal(9999))
					Expect(cfg.LogGuid).To(Equal("my_logs"))
					Expect(cfg.MetronConfig.Address).To(Equal("1.2.3.4"))
					Expect(cfg.MetronConfig.Port).To(Equal("4567"))
					Expect(cfg.StatsdClientFlushInterval).To(Equal(10 * time.Millisecond))
					Expect(cfg.OAuth.TokenEndpoint).To(Equal("127.0.0.1"))
					Expect(cfg.OAuth.Port).To(Equal(8080))
					Expect(cfg.OAuth.CACerts).To(Equal("some-ca-cert"))
					Expect(cfg.OAuth.SkipSSLValidation).To(Equal(true))
					Expect(cfg.SystemDomain).To(Equal("example.com"))
					Expect(cfg.SqlDB.Username).To(Equal("username"))
					Expect(cfg.SqlDB.Password).To(Equal("password"))
					Expect(cfg.SqlDB.Port).To(Equal(1234))
					Expect(cfg.SqlDB.CACert).To(Equal("some CA cert"))
					Expect(cfg.SqlDB.SkipSSLValidation).To(Equal(true))
					Expect(cfg.SqlDB.SkipHostnameValidation).To(Equal(true))
					Expect(cfg.MaxTTL).To(Equal(2 * time.Minute))
					Expect(cfg.ConsulCluster.Servers).To(Equal("http://localhost:5678"))
					Expect(cfg.ConsulCluster.LockTTL).To(Equal(10 * time.Second))
					Expect(cfg.ConsulCluster.RetryInterval).To(Equal(5 * time.Second))
					Expect(cfg.SkipConsulLock).To(BeTrue())
					Expect(cfg.Locket.LocketAddress).To(Equal("http://localhost:5678"))
					Expect(cfg.Locket.LocketCACertFile).To(Equal("some-locket-ca-cert"))
					Expect(cfg.Locket.LocketClientCertFile).To(Equal("some-locket-client-cert"))
					Expect(cfg.Locket.LocketClientKeyFile).To(Equal("some-locket-client-key"))
				})

				Context("when there is no token endpoint specified", func() {
					It("returns an error", func() {
						cfg_file := "../example_config/missing_uaa_url.yml"
						_, err := config.NewConfigFromFile(cfg_file, false)
						Expect(err).To(HaveOccurred())
					})
				})
			})

			Context("when the file does not exists", func() {
				It("returns an error", func() {
					cfg_file := "notexist"
					_, err := config.NewConfigFromFile(cfg_file, false)

					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("when auth is disabled", func() {
			Context("when the file exists", func() {
				It("returns a valid config", func() {
					cfg_file := "../example_config/example.yml"
					cfg, err := config.NewConfigFromFile(cfg_file, true)

					Expect(err).NotTo(HaveOccurred())
					Expect(cfg.LogGuid).To(Equal("my_logs"))
					Expect(cfg.MetronConfig.Address).To(Equal("1.2.3.4"))
					Expect(cfg.MetronConfig.Port).To(Equal("4567"))
					Expect(cfg.StatsdClientFlushInterval).To(Equal(10 * time.Millisecond))
					Expect(cfg.OAuth.TokenEndpoint).To(Equal("127.0.0.1"))
					Expect(cfg.OAuth.Port).To(Equal(8080))
					Expect(cfg.OAuth.CACerts).To(Equal("some-ca-cert"))
				})

				Context("when there is no token endpoint url", func() {
					It("returns a valid config", func() {
						cfg_file := "../example_config/missing_uaa_url.yml"
						cfg, err := config.NewConfigFromFile(cfg_file, true)

						Expect(err).NotTo(HaveOccurred())
						Expect(cfg.LogGuid).To(Equal("my_logs"))
						Expect(cfg.MetronConfig.Address).To(Equal("1.2.3.4"))
						Expect(cfg.MetronConfig.Port).To(Equal("4567"))
						Expect(cfg.DebugAddress).To(Equal("1.2.3.4:1234"))
						Expect(cfg.MaxTTL).To(Equal(2 * time.Minute))
						Expect(cfg.StatsdClientFlushInterval).To(Equal(10 * time.Millisecond))
						Expect(cfg.OAuth.TokenEndpoint).To(BeEmpty())
						Expect(cfg.OAuth.Port).To(Equal(0))
					})
				})
			})
		})
	})

	Describe("parsing and validating the configuration", func() {
		Context("when UUID property is set", func() {
			It("populates the value", func() {
				testConfig := `log_guid: "my_logs"
admin_port: 9999
api:
  listen_port: 3000
metrics_reporting_interval: "500ms"
uuid: "fake-uuid"
statsd_endpoint: "localhost:8125"
statsd_client_flush_interval: "10ms"
system_domain: "example.com"
router_groups:
- name: router-group-2
  reservable_ports: 1024-10000,42000
  type: udp
consul_cluster:
  url: "http://localhost:4222"
`
				cfg, err := config.NewConfigFromBytes([]byte(testConfig), true)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.UUID).To(Equal("fake-uuid"))
			})
		})

		Context("when the api listen port is invalid", func() {
			testConfig := func(apiPort int) string {
				return fmt.Sprintf(`log_guid: "my_logs"
admin_port: 9999
api:
  listen_port: %d
metrics_reporting_interval: "500ms"
uuid: "fake-uuid"
statsd_endpoint: "localhost:8125"
statsd_client_flush_interval: "10ms"
system_domain: "example.com"
router_groups:
- name: router-group-2
  reservable_ports: 1024-10000,42000
  type: udp
consul_cluster:
  url: "http://localhost:4222"
`, apiPort)
			}
			Context("when it is too high", func() {
				It("returns an error", func() {
					_, err := config.NewConfigFromBytes([]byte(testConfig(65535)), true)
					Expect(err).NotTo(HaveOccurred())

					_, err = config.NewConfigFromBytes([]byte(testConfig(65536)), true)
					Expect(err).To(HaveOccurred())
				})
			})

			Context("when it is too low", func() {
				It("returns an error", func() {
					_, err := config.NewConfigFromBytes([]byte(testConfig(0)), true)
					Expect(err).To(HaveOccurred())

					_, err = config.NewConfigFromBytes([]byte(testConfig(1)), true)
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})

		Context("when UUID property is not set", func() {
			It("populates the value", func() {
				testConfig :=
					`log_guid: "my_logs"
admin_port: 9999
api:
  listen_port: 3000
metrics_reporting_interval: "500ms"
statsd_endpoint: "localhost:8125"
statsd_client_flush_interval: "10ms"
system_domain: "example.com"
router_groups:
- name: router-group-2
  reservable_ports: 1024-10000,42000
  type: udp
consul_cluster:
  url: "http://localhost:4222"
`
				_, err := config.NewConfigFromBytes([]byte(testConfig), true)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(errors.New("No UUID is specified")))
			})
		})

		Context("when AdminPort property is set", func() {
			It("populates the value", func() {
				testConfig := `admin_port: 9999
log_guid: "my_logs"
api:
  listen_port: 3000
metrics_reporting_interval: "500ms"
uuid: "fake-uuid"
statsd_endpoint: "localhost:8125"
statsd_client_flush_interval: "10ms"
system_domain: "example.com"
router_groups:
- name: router-group-2
  reservable_ports: 1024-10000,42000
  type: udp
consul_cluster:
  url: "http://localhost:4222"
`
				cfg, err := config.NewConfigFromBytes([]byte(testConfig), true)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.AdminPort).To(Equal(9999))
			})
		})

		Context("when AdminPort property is not set", func() {
			It("returns an error", func() {
				testConfig := `log_guid: "my_logs"
metrics_reporting_interval: "500ms"
uuid: "fake-uuid"
api:
  listen_port: 3000
statsd_endpoint: "localhost:8125"
statsd_client_flush_interval: "10ms"
system_domain: "example.com"
router_groups:
- name: router-group-2
  reservable_ports: 1024-10000,42000
  type: udp
consul_cluster:
  url: "http://localhost:4222"
`
				_, err := config.NewConfigFromBytes([]byte(testConfig), true)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when consul properties are not set", func() {
			var testConfig string

			BeforeEach(func() {
				testConfig = `log_guid: "my_logs"
admin_port: 9999
api:
  listen_port: 3000
metrics_reporting_interval: "500ms"
statsd_endpoint: "localhost:8125"
statsd_client_flush_interval: "10ms"
uuid: "fake-uuid"
system_domain: "example.com"
router_groups:
- name: router-group-2
  reservable_ports: 1024-10000,42000
  type: udp
consul_cluster:
  url: "http://localhost:4222"
`
			})

			It("populates the default value for LockTTL from locket library", func() {
				cfg, err := config.NewConfigFromBytes([]byte(testConfig), true)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.ConsulCluster.LockTTL).To(Equal(locket.DefaultSessionTTL))
			})

			It("populates the default value for RetryInterval from locket library", func() {
				cfg, err := config.NewConfigFromBytes([]byte(testConfig), true)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.ConsulCluster.RetryInterval).To(Equal(locket.RetryInterval))
			})
		})

		Context("when multiple router groups are seeded with different names", func() {
			It("should not error", func() {
				testConfigStr := `log_guid: "my_logs"
admin_port: 9999
api:
  listen_port: 3000
metrics_reporting_interval: "500ms"
statsd_endpoint: "localhost:8125"
statsd_client_flush_interval: "10ms"
uuid: "fake-uuid"
system_domain: "example.com"
router_groups:
- name: router-group-1
  reservable_ports: 1200
  type: tcp
- name: router-group-2
  reservable_ports: 10000-42000
  type: tcp`

				cfg, err := config.NewConfigFromBytes([]byte(testConfigStr), true)
				Expect(err).NotTo(HaveOccurred())
				expectedGroups := models.RouterGroups{
					{
						Name:            "router-group-1",
						ReservablePorts: "1200",
						Type:            "tcp",
					},
					{
						Name:            "router-group-2",
						ReservablePorts: "10000-42000",
						Type:            "tcp",
					},
				}
				Expect(cfg.RouterGroups).To(Equal(expectedGroups))
			})
		})

		Context("when router groups are seeded in the configuration file", func() {
			var expectedGroups models.RouterGroups

			testConfig := func(ports string) string {
				return `log_guid: "my_logs"
admin_port: 9999
api:
  listen_port: 3000
metrics_reporting_interval: "500ms"
statsd_endpoint: "localhost:8125"
statsd_client_flush_interval: "10ms"
uuid: "fake-uuid"
system_domain: "example.com"
router_groups:
- name: router-group-1
  reservable_ports: ` + ports + `
  type: tcp
- name: router-group-2
  reservable_ports: 1024-10000,42000
  type: udp
- name: router-group-special
  reservable_ports:
  - 1122
  - 1123
  type: tcp`
			}

			It("populates the router groups", func() {
				configStr := testConfig("12000")
				cfg, err := config.NewConfigFromBytes([]byte(configStr), true)
				Expect(err).NotTo(HaveOccurred())
				expectedGroups = models.RouterGroups{
					{
						Name:            "router-group-1",
						ReservablePorts: "12000",
						Type:            "tcp",
					},
					{
						Name:            "router-group-2",
						ReservablePorts: "1024-10000,42000",
						Type:            "udp",
					},
					{
						Name:            "router-group-special",
						ReservablePorts: "1122,1123",
						Type:            "tcp",
					},
				}
				Expect(cfg.RouterGroups).To(Equal(expectedGroups))
			})

			It("returns an error when port array has invalid type", func() {
				configStr := `log_guid: "my_logs"
admin_port: 9999
api:
  listen_port: 3000
metrics_reporting_interval: "500ms"
statsd_endpoint: "localhost:8125"
statsd_client_flush_interval: "10ms"
uuid: "fake-uuid"
system_domain: "example.com"
router_groups:
- name: router-group-special
  reservable_ports:
  - "1122"
  - 1123
  type: tcp`
				_, err := config.NewConfigFromBytes([]byte(configStr), true)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid type for reservable port"))
			})

			It("returns error for invalid ports", func() {
				configStr := testConfig("abc")
				_, err := config.NewConfigFromBytes([]byte(configStr), true)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Port must be between 1024 and 65535"))
			})

			It("does not returns error for ports prefixed with zero", func() {
				configStr := testConfig("00003202-4000")
				_, err := config.NewConfigFromBytes([]byte(configStr), true)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns error for invalid port", func() {
				configStr := testConfig("70000")
				_, err := config.NewConfigFromBytes([]byte(configStr), true)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Port must be between 1024 and 65535"))
			})

			It("returns error for invalid ranges of ports", func() {
				configStr := testConfig("1024-65535,10000-20000")
				_, err := config.NewConfigFromBytes([]byte(configStr), true)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Overlapping values: [1024-65535] and [10000-20000]"))
			})

			It("returns error for invalid range of ports", func() {
				configStr := testConfig("1023-65530")
				_, err := config.NewConfigFromBytes([]byte(configStr), true)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Port must be between 1024 and 65535"))
			})

			It("returns error for invalid start range", func() {
				configStr := testConfig("1024-65535,-10000")
				_, err := config.NewConfigFromBytes([]byte(configStr), true)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("range (-10000) requires a starting port"))
			})

			It("returns error for invalid end range", func() {
				configStr := testConfig("10000-")
				_, err := config.NewConfigFromBytes([]byte(configStr), true)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("range (10000-) requires an ending port"))
			})

			It("returns error for invalid router group type", func() {
				missingType := `log_guid: "my_logs"
admin_port: 9999
api:
  listen_port: 3000
metrics_reporting_interval: "500ms"
statsd_endpoint: "localhost:8125"
statsd_client_flush_interval: "10ms"
uuid: "fake-uuid"
system_domain: "example.com"
router_groups:
- name: router-group-1
  reservable_ports: 1024-65535`
				_, err := config.NewConfigFromBytes([]byte(missingType), true)
				Expect(err).To(HaveOccurred())
			})

			It("returns error for invalid router group type", func() {
				missingName := `log_guid: "my_logs"
admin_port: 9999
api:
  listen_port: 3000
metrics_reporting_interval: "500ms"
statsd_endpoint: "localhost:8125"
statsd_client_flush_interval: "10ms"
uuid: "fake-uuid"
system_domain: "example.com"
router_groups:
- type: tcp
  reservable_ports: 1024-65535`
				_, err := config.NewConfigFromBytes([]byte(missingName), true)
				Expect(err).To(HaveOccurred())
			})

			It("returns error for missing reservable port range", func() {
				missingRouterGroup := `log_guid: "my_logs"
admin_port: 9999
api:
  listen_port: 3000
metrics_reporting_interval: "500ms"
statsd_endpoint: "localhost:8125"
statsd_client_flush_interval: "10ms"
uuid: "fake-uuid"
system_domain: "example.com"
router_groups:
- type: tcp
  name: default-tcp`
				_, err := config.NewConfigFromBytes([]byte(missingRouterGroup), true)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Missing reservable_ports in router group:"))
			})
		})

		Context("when there are errors in the yml file", func() {
			It("errors if there is no system_domain", func() {
				testConfig := `log_guid: "my_logs"
admin_port: 9999
api:
  listen_port: 3000
debug_address: "1.2.3.4:1234"
metron_config:
  address: "1.2.3.4"
  port: "4567"
metrics_reporting_interval: "500ms"
uuid: "fake-uuid"
statsd_endpoint: "localhost:8125"
statsd_client_flush_interval: "10ms"`
				_, err := config.NewConfigFromBytes([]byte(testConfig), true)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("system_domain"))
			})

			Context("UAA errors", func() {
				var testConfig string
				BeforeEach(func() {
					testConfig = `log_guid: "my_logs"
admin_port: 9999
api:
  listen_port: 3000
debug_address: "1.2.3.4:1234"
system_domain: "example.com"
uuid: "fake-uuid"
metron_config:
  address: "1.2.3.4"
  port: "4567"
metrics_reporting_interval: "500ms"
statsd_endpoint: "localhost:8125"
statsd_client_flush_interval: "10ms"`
				})

				Context("when auth is enabled", func() {
					It("errors if no token endpoint url is found", func() {
						_, err := config.NewConfigFromBytes([]byte(testConfig), false)
						Expect(err).To(HaveOccurred())
					})
				})

				Context("when auth is disabled", func() {
					It("it return valid config", func() {
						_, err := config.NewConfigFromBytes([]byte(testConfig), true)
						Expect(err).NotTo(HaveOccurred())
					})
				})
			})
		})

		Context("when there are no router groups seeded in the configuration file", func() {
			It("does not populates the router group", func() {
				testConfig := `log_guid: "my_logs"
admin_port: 9999
api:
  listen_port: 3000
system_domain: "example.com"
metrics_reporting_interval: "500ms"
uuid: "fake-uuid"
statsd_endpoint: "localhost:8125"
statsd_client_flush_interval: "10ms"`
				cfg, err := config.NewConfigFromBytes([]byte(testConfig), true)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.RouterGroups).To(BeNil())
			})

		})
	})
})
