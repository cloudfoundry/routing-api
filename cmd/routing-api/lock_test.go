package main_test

import (
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Locking", func() {
	BeforeEach(func() {
		oauthServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", TOKEN_KEY_ENDPOINT),
				ghttp.RespondWith(http.StatusOK, `{"alg":"rsa", "value":
				"-----BEGIN RSA PUBLIC KEY-----MIICCgKCAgEAvXi0gTxLcrNJrRTjKu45UdhCQyHDhQddPnA5bIr2ofdZYogx4K/naFc0rbfEIboGsOH+Tj02ku1j+rEqDqT2tbJlKg5NzRrlXLnBolHCTLjHernSJ7LiO/p30bkCaqlAQPVFayPovcJPH9ONSnFe8YqO08cxG/qvARULEPnAJt9Ciijh8uzVBpSGrk8bNeN6cqlIWwUmHe6HDbwn3X1zGnuX1pHtXLzXXeUASqj0I2BQy/JgsGJEsnZ54XKzq0MehEdXsjqX4NKm6Eab1lPxMOhNB4jsR/agXMDRcYk6IUIh4Oz35JddN14nQxiphgcsAgLuL+f+TQvMREHvNBNuGnCMZFlu/B7EFKPaRXrYJ0XX7OZw4sByc8CogwWMLdM6fivkSw2yW2nWJUVwktauWcJ1RT9DklmB8ABcAXddnV/S4hdQxLNNUV0sP1na9oO8CBudgFVA19tXj9mXfyK7YFNE/t1hGRdtZGJyRADc7M1KvTSrHBgowO6o4n2vQ+7spOb7Klei7ZS7z+a30zbom/IvZLTWdAh/1D+zAlIk9Fj6YDOVTjvha5+WCInACTPWapY2Ed1pFYail1UesgBK1N6aJrgK0f5YY2mtH+BfsSjagqHU3Ax15y85RunQX6nhths1gfjf5D4SvjH1BJ4AVGNP6tDdw/Mx0GghlS1fPzMCAwEAAQ==-----END RSA PUBLIC KEY-----"}`),
			),
		)
	})
	Describe("vieing for the lock", func() {
		Context("when two long-lived processes try to run", func() {
			It("one waits for the other to exit and then grabs the lock", func() {
				args := routingAPIArgs
				args.DevMode = true
				session1 := RoutingApi(args.ArgSlice()...)
				Eventually(session1, 10*time.Second).Should(gbytes.Say("acquire-lock-succeeded"))

				session2 := RoutingApi(args.ArgSlice()...)

				defer func() {
					session1.Interrupt().Wait(5 * time.Second)
					session2.Interrupt().Wait(5 * time.Second)
				}()

				Eventually(session2, 10*time.Second).Should(gbytes.Say("acquiring-lock"))
				Consistently(session2).ShouldNot(gbytes.Say("acquire-lock-succeeded"))

				session1.Interrupt().Wait(5 * time.Second)

				Eventually(session1, 10*time.Second).Should(gbytes.Say("releasing-lock"))
				Eventually(session2, 10*time.Second).Should(gbytes.Say("acquire-lock-succeeded"))
			})
		})

	})

	Context("when the lock disappears", func() {
		Context("long-lived processes", func() {
			It("should exit 197", func() {
				args := routingAPIArgs
				args.DevMode = true
				session1 := RoutingApi(args.ArgSlice()...)
				defer func() {
					session1.Interrupt().Wait(5 * time.Second)
				}()

				Eventually(session1, 10*time.Second).Should(gbytes.Say("acquire-lock-succeeded"))

				consulRunner.Reset()
				consulRunner.WaitUntilReady()
				Eventually(session1, 10*time.Second).Should(gbytes.Say("lost-lock"))
				Eventually(session1, 20*time.Second).Should(gexec.Exit(1))
			})
		})
	})
})
