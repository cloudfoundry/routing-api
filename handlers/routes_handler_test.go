package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/pivotal-cf-experimental/routing-api/db"
	"github.com/pivotal-cf-experimental/routing-api/db/fakes"
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
		Ω(err).ToNot(HaveOccurred())
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
		database         *fakes.FakeDB
		logger           *lagertest.TestLogger
	)

	BeforeEach(func() {
		database = &fakes.FakeDB{}
		logger = lagertest.NewTestLogger("routing-api-test")
		routesHandler = handlers.NewRoutesHandler(50, database, logger)
		responseRecorder = httptest.NewRecorder()
	})

	Describe(".DeleteRoute", func() {
		var (
			route db.Route
		)

		BeforeEach(func() {
			route = db.Route{
				Route: "post_here",
				IP:    "1.2.3.4",
				Port:  7000,
			}
		})

		Context("when all inputs are present and correct", func() {
			It("returns a status not found when deleting a route", func() {
				request = newTestRequest(route)

				routesHandler.Delete(responseRecorder, request)

				Expect(responseRecorder.Code).To(Equal(http.StatusNoContent))
				Expect(database.DeleteRouteCallCount()).To(Equal(1))
				Expect(database.DeleteRouteArgsForCall(0)).To(Equal(route))
			})

			It("logs the route deletion", func() {
				request = newTestRequest(route)
				routesHandler.Delete(responseRecorder, request)

				Expect(logger.Logs()[0].Message).To(ContainSubstring("request"))
				Expect(logger.Logs()[0].Data["route_deletion"]).To(Equal(map[string]interface{}{
					"ip":       "1.2.3.4",
					"log_guid": "",
					"port":     float64(7000),
					"route":    "post_here",
					"ttl":      float64(0),
				}))
			})

			Context("when the database deletion fails", func() {
				BeforeEach(func() {
					database.DeleteRouteReturns(errors.New("stuff broke"))
				})

				It("responds with a server error", func() {
					request = newTestRequest(route)
					routesHandler.Delete(responseRecorder, request)

					Expect(responseRecorder.Code).To(Equal(http.StatusInternalServerError))
					Expect(responseRecorder.Body.String()).To(ContainSubstring("stuff broke"))
				})
			})
		})

		Context("when there are errors with the input", func() {
			It("returns a bad request if it cannot parse the arguments", func() {
				request = newTestRequest("bad args")

				routesHandler.Delete(responseRecorder, request)

				Expect(responseRecorder.Code).To(Equal(http.StatusBadRequest))
				Expect(responseRecorder.Body.String()).To(ContainSubstring("Cannot process request"))
			})

			It("returns invalid request if there is no route in the body", func() {
				route.Route = ""

				request = newTestRequest(route)
				routesHandler.Delete(responseRecorder, request)

				Expect(responseRecorder.Code).To(Equal(http.StatusBadRequest))
				Expect(responseRecorder.Body.String()).To(ContainSubstring("Request requires a route"))
				Expect(database.DeleteRouteCallCount()).To(Equal(0))
			})

			It("returns invalid request if the port is less than 1", func() {
				route.Port = 0

				request = newTestRequest(route)
				routesHandler.Delete(responseRecorder, request)

				Expect(responseRecorder.Code).To(Equal(http.StatusBadRequest))
				Expect(responseRecorder.Body.String()).To(ContainSubstring("Request requires a port greater than 0"))
				Expect(database.DeleteRouteCallCount()).To(Equal(0))
			})

			It("returns invalid request if there is no IP in the body", func() {
				route.IP = ""

				request = newTestRequest(route)
				routesHandler.Delete(responseRecorder, request)

				Expect(responseRecorder.Code).To(Equal(http.StatusBadRequest))
				Expect(responseRecorder.Body.String()).To(ContainSubstring("Request requires a valid ip"))
				Expect(database.DeleteRouteCallCount()).To(Equal(0))
			})

		})
	})

	Describe(".Routes", func() {
		Context("POST", func() {
			var (
				route db.Route
			)

			BeforeEach(func() {
				route = db.Route{
					Route: "post_here",
					IP:    "1.2.3.4",
					Port:  7000,
					TTL:   50,
				}
			})

			Context("when all inputs are present and correct", func() {
				It("returns an http status created", func() {
					request = newTestRequest(route)
					routesHandler.Routes(responseRecorder, request)

					Expect(responseRecorder.Code).To(Equal(http.StatusCreated))
				})

				It("logs the route declaration", func() {
					request = newTestRequest(route)
					routesHandler.Routes(responseRecorder, request)

					Expect(logger.Logs()[0].Message).To(ContainSubstring("request"))
					Expect(logger.Logs()[0].Data["route_declaration"]).To(Equal(map[string]interface{}{
						"ip":       "1.2.3.4",
						"log_guid": "",
						"port":     float64(7000),
						"route":    "post_here",
						"ttl":      float64(50),
					}))
				})

				It("does not require log guid on the request", func() {
					route.LogGuid = ""

					request = newTestRequest(route)
					routesHandler.Routes(responseRecorder, request)

					Expect(responseRecorder.Code).To(Equal(http.StatusCreated))
				})

				It("writes to database backend", func() {
					route.LogGuid = "my-guid"

					request = newTestRequest(route)
					routesHandler.Routes(responseRecorder, request)

					Expect(database.SaveRouteCallCount()).To(Equal(1))
					Expect(database.SaveRouteArgsForCall(0)).To(Equal(route))
				})

				Context("when database fails to save", func() {
					BeforeEach(func() {
						database.SaveRouteReturns(errors.New("stuff broke"))
					})

					It("responds with a server error", func() {
						request = newTestRequest(route)
						routesHandler.Routes(responseRecorder, request)

						Expect(responseRecorder.Code).To(Equal(http.StatusInternalServerError))
						Expect(responseRecorder.Body.String()).To(ContainSubstring("stuff broke"))
					})
				})
			})

			Context("when there are errors with the input", func() {
				It("does not write to the key-value store backend", func() {
					route.Route = ""

					request = newTestRequest(route)
					routesHandler.Routes(responseRecorder, request)

					Expect(database.SaveRouteCallCount()).To(Equal(0))
				})

				It("logs the error", func() {
					route.Route = ""

					request = newTestRequest(route)
					routesHandler.Routes(responseRecorder, request)

					Expect(logger.Logs()[1].Message).To(ContainSubstring("error"))
					Expect(logger.Logs()[1].Data["error"]).To(Equal("Request requires a route"))
				})

				It("rejects if too high of a ttl", func() {
					routesHandler = handlers.NewRoutesHandler(47, database, logger)
					request = newTestRequest(db.Route{
						Route: "post_here",
						IP:    "1.2.3.4",
						Port:  7000,
						TTL:   49,
					})

					routesHandler.Routes(responseRecorder, request)

					Expect(responseRecorder.Code).To(Equal(http.StatusBadRequest))
					Expect(responseRecorder.Body.String()).To(ContainSubstring("Max ttl is 47"))
				})

				It("returns invalid request if there is no route in the body", func() {
					route.Route = ""

					request = newTestRequest(route)
					routesHandler.Routes(responseRecorder, request)

					Expect(responseRecorder.Code).To(Equal(http.StatusBadRequest))
					Expect(responseRecorder.Body.String()).To(ContainSubstring("Request requires a route"))
				})

				It("returns invalid request if the port is less than 1", func() {
					route.Port = 0

					request = newTestRequest(route)
					routesHandler.Routes(responseRecorder, request)

					Expect(responseRecorder.Code).To(Equal(http.StatusBadRequest))
					Expect(responseRecorder.Body.String()).To(ContainSubstring("Request requires a port greater than 0"))
				})

				It("returns invalid request if there is no IP in the body", func() {
					route.IP = ""

					request = newTestRequest(route)
					routesHandler.Routes(responseRecorder, request)

					Expect(responseRecorder.Code).To(Equal(http.StatusBadRequest))
					Expect(responseRecorder.Body.String()).To(ContainSubstring("Request requires a valid ip"))
				})

				It("returns invalid request if the ttl is less than 1 in the body", func() {
					route.TTL = 0

					request = newTestRequest(route)
					routesHandler.Routes(responseRecorder, request)

					Expect(responseRecorder.Code).To(Equal(http.StatusBadRequest))
					Expect(responseRecorder.Body.String()).To(ContainSubstring("Request requires a ttl greater than 0"))
				})
			})
		})
	})
})
