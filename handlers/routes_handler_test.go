package handlers_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/pivotal-cf-experimental/routing-api/handlers"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func newTestRequest(body interface{}) *http.Request {
	var reader io.Reader
	switch body := body.(type) {

	case string:
		reader = strings.NewReader(body)
	case []byte:
		reader = bytes.NewReader(body)
	default:
		jsonBytes, err := json.Marshal(body)
		Ω(err).ShouldNot(HaveOccurred())
		reader = bytes.NewReader(jsonBytes)
	}

	request, err := http.NewRequest("", "", reader)
	Ω(err).ToNot(HaveOccurred())
	return request
}

var _ = Describe("RoutesHandler", func() {
	var (
		routesHandler    *handlers.RoutesHandler
		request          *http.Request
		responseRecorder *httptest.ResponseRecorder
		logger           *lagertest.TestLogger
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("routing-api-test")
		routesHandler = handlers.NewRoutesHandler(50, logger)
		responseRecorder = httptest.NewRecorder()
	})

	Describe(".Routes", func() {
		Context("POST", func() {
			var (
				routeDeclaration handlers.RouteDeclaration
			)

			BeforeEach(func() {
				routeDeclaration = handlers.RouteDeclaration{
					Route: "/post_here",
					TTL:   50,
					IP:    "1.2.3.4",
					Port:  7000,
				}
			})

			It("logs the route declaration", func() {
				request = newTestRequest(routeDeclaration)
				routesHandler.Routes(responseRecorder, request)

				Expect(logger.Logs()[0].Message).To(ContainSubstring("request"))
				Expect(logger.Logs()[0].Data["route_declaration"]).To(Equal(map[string]interface{}{
					"ip":       "1.2.3.4",
					"log_guid": "",
					"port":     float64(7000),
					"route":    "/post_here",
					"ttl":      float64(50),
				}))
			})

			It("returns an http status created", func() {
				request = newTestRequest(routeDeclaration)
				routesHandler.Routes(responseRecorder, request)

				Expect(responseRecorder.Code).Should(Equal(http.StatusCreated))
			})

			It("logs the error", func() {
				routeDeclaration.Route = ""

				request = newTestRequest(routeDeclaration)
				routesHandler.Routes(responseRecorder, request)

				Expect(logger.Logs()[1].Message).To(ContainSubstring("error"))
				Expect(logger.Logs()[1].Data["error"]).To(Equal("Request requires a route"))
			})

			It("rejects if too high of a ttl", func() {
				routesHandler = handlers.NewRoutesHandler(47, logger)
				request = newTestRequest(handlers.RouteDeclaration{
					Route: "/post_here",
					TTL:   49,
					IP:    "1.2.3.4",
					Port:  7000,
				})

				routesHandler.Routes(responseRecorder, request)

				Expect(responseRecorder.Code).Should(Equal(http.StatusBadRequest))
				Expect(string(responseRecorder.Body.String())).Should(Equal("Max ttl is 47"))
			})

			It("returns invalid request if there is no route in the body", func() {
				routeDeclaration.Route = ""

				request = newTestRequest(routeDeclaration)
				routesHandler.Routes(responseRecorder, request)

				Expect(responseRecorder.Code).Should(Equal(http.StatusBadRequest))
				Expect(string(responseRecorder.Body.String())).Should(Equal("Request requires a route"))
			})

			It("returns invalid request if the port is less than 1", func() {
				routeDeclaration.Port = 0

				request = newTestRequest(routeDeclaration)
				routesHandler.Routes(responseRecorder, request)

				Expect(responseRecorder.Code).Should(Equal(http.StatusBadRequest))
				Expect(string(responseRecorder.Body.String())).Should(Equal("Request requires a port greater than 0"))
			})

			It("returns invalid request if there is no IP in the body", func() {
				routeDeclaration.IP = ""

				request = newTestRequest(routeDeclaration)
				routesHandler.Routes(responseRecorder, request)

				Expect(responseRecorder.Code).Should(Equal(http.StatusBadRequest))
				Expect(string(responseRecorder.Body.String())).Should(Equal("Request requires a valid ip"))
			})

			It("returns invalid request if the ttl is less than 1 in the body", func() {
				routeDeclaration.TTL = 0

				request = newTestRequest(routeDeclaration)
				routesHandler.Routes(responseRecorder, request)

				Expect(responseRecorder.Code).Should(Equal(http.StatusBadRequest))
				Expect(string(responseRecorder.Body.String())).Should(Equal("Request requires a ttl greater than 0"))
			})

			It("does not require log guid on the request", func() {
				routeDeclaration.LogGuid = ""

				request = newTestRequest(routeDeclaration)
				routesHandler.Routes(responseRecorder, request)

				Expect(responseRecorder.Code).To(Equal(http.StatusCreated))
			})
		})
	})
})
