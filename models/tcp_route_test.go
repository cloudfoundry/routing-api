package models_test

import (
	"encoding/json"

	"code.cloudfoundry.org/routing-api/models"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func pointerToString(s string) *string { return &s }

var _ = Describe("TCP Route", func() {
	Describe("TcpMappingEntity", func() {
		var tcpRouteMapping models.TcpRouteMapping
		var sniHostNamePtr *string

		JustBeforeEach(func() {
			tcpRouteMapping = models.NewTcpRouteMapping("a-guid", 1234, "hostIp", 5678, 8765, "", sniHostNamePtr, 5, models.ModificationTag{})
		})
		Describe("SNI Hostname", func() {
			Context("when the SNI hostname is nil", func() {
				BeforeEach(func() {
					sniHostNamePtr = nil
				})
				It("comes through as nil", func() {
					Expect(tcpRouteMapping.SniHostname).To(BeNil())
				})
				It("is omitted from JSON  marshaling", func() {
					j, err := json.Marshal(tcpRouteMapping)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(j)).NotTo(ContainSubstring("backend_sni_hostname"))
				})
			})

			Context("when a valid SNI hostname is provided", func() {
				BeforeEach(func() {
					sniHostNamePtr = pointerToString("sniHostname")
				})

				It("Accepts the value", func() {
					Expect(*tcpRouteMapping.SniHostname).To(Equal("sniHostname"))
				})
				It("is provided in the marshaled JSON", func() {
					j, err := json.Marshal(tcpRouteMapping)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(j)).To(ContainSubstring("backend_sni_hostname"))
				})
			})
			Context("when the SNI hostname is empty", func() {
				BeforeEach(func() {
					sniHostNamePtr = pointerToString("")
				})
				It("is provided in the marshaled JSON", func() {
					j, err := json.Marshal(tcpRouteMapping)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(j)).To(ContainSubstring("backend_sni_hostname"))
				})
			})
		})
		Describe("Matches()", func() {
			var tcpRouteMapping2 models.TcpRouteMapping
			var sniHostNamePtr2 *string

			BeforeEach(func() {
				sniHostNamePtr = pointerToString("sniHostName")
			})

			JustBeforeEach(func() {
				tcpRouteMapping2 = models.NewTcpRouteMapping("a-guid", 1234, "hostIp", 5678, 8765, "", sniHostNamePtr2, 5, models.ModificationTag{})
			})

			Context("when two routes have the same SNIHostName value", func() {
				BeforeEach(func() {
					sniHostNamePtr2 = sniHostNamePtr
				})
				It("matches", func() {
					Expect(tcpRouteMapping.Matches(tcpRouteMapping2)).To(BeTrue())
				})
			})
			Context("when two routes have equal values", func() {
				BeforeEach(func() {
					sniHostNamePtr2 = pointerToString("sniHostName")
				})
				It("matches", func() {
					Expect(tcpRouteMapping.Matches(tcpRouteMapping2)).To(BeTrue())
				})
			})

			Context("when two routes have values that are not equal", func() {
				BeforeEach(func() {
					sniHostNamePtr2 = pointerToString("sniHostName2")
				})
				It("doesn't match", func() {
					Expect(tcpRouteMapping.Matches(tcpRouteMapping2)).To(BeFalse())
				})
			})
			Context("when one of the routes has a nil SNIHostName", func() {
				BeforeEach(func() {
					sniHostNamePtr2 = nil
				})
				It("doesn't match", func() {
					Expect(tcpRouteMapping.Matches(tcpRouteMapping2)).To(BeFalse())
					Expect(tcpRouteMapping2.Matches(tcpRouteMapping)).To(BeFalse())
				})
			})
		})
	})
})
