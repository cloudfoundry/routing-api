package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	fake_token "github.com/pivotal-cf-experimental/routing-api/authentication/fakes"
	"github.com/pivotal-cf-experimental/routing-api/db"
	fake_db "github.com/pivotal-cf-experimental/routing-api/db/fakes"
	"github.com/pivotal-cf-experimental/routing-api/handlers"
	fake_validator "github.com/pivotal-cf-experimental/routing-api/handlers/fakes"
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
		database         *fake_db.FakeDB
		logger           *lagertest.TestLogger
		validator        *fake_validator.FakeRouteValidator
		token            *fake_token.FakeToken
	)

	BeforeEach(func() {
		database = &fake_db.FakeDB{}
		validator = &fake_validator.FakeRouteValidator{}
		token = &fake_token.FakeToken{}
		logger = lagertest.NewTestLogger("routing-api-test")
		routesHandler = handlers.NewRoutesHandler(token, 50, validator, database, logger)
		responseRecorder = httptest.NewRecorder()
	})

	Describe(".DeleteRoute", func() {
		var (
			route []db.Route
		)

		BeforeEach(func() {
			route = []db.Route{
				{
					Route: "post_here",
					IP:    "1.2.3.4",
					Port:  7000,
				},
			}
		})

		Context("when all inputs are present and correct", func() {
			It("returns a status not found when deleting a route", func() {
				request = newTestRequest(route)

				routesHandler.Delete(responseRecorder, request)

				Expect(responseRecorder.Code).To(Equal(http.StatusNoContent))
				Expect(database.DeleteRouteCallCount()).To(Equal(1))
				Expect(database.DeleteRouteArgsForCall(0)).To(Equal(route[0]))
			})

			It("accepts an array of routes in the body", func() {
				route = append(route, route[0])
				route[1].IP = "5.4.3.2"

				request = newTestRequest(route)
				routesHandler.Delete(responseRecorder, request)

				Expect(responseRecorder.Code).To(Equal(http.StatusNoContent))
				Expect(database.DeleteRouteCallCount()).To(Equal(2))
				Expect(database.DeleteRouteArgsForCall(0)).To(Equal(route[0]))
				Expect(database.DeleteRouteArgsForCall(1)).To(Equal(route[1]))
			})

			It("logs the route deletion", func() {
				request = newTestRequest(route)
				routesHandler.Delete(responseRecorder, request)

				data := map[string]interface{}{
					"ip":       "1.2.3.4",
					"log_guid": "",
					"port":     float64(7000),
					"route":    "post_here",
					"ttl":      float64(0),
				}
				log_data := map[string][]interface{}{"route_deletion": []interface{}{data}}

				Expect(logger.Logs()[0].Message).To(ContainSubstring("request"))
				Expect(logger.Logs()[0].Data["route_deletion"]).To(Equal(log_data["route_deletion"]))
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
		})

		Context("when the UAA token is not valid", func() {
			BeforeEach(func() {
				token.DecodeTokenReturns(errors.New("Not valid"))
			})

			It("returns an Unauthorized status code", func() {
				request = newTestRequest(route)
				routesHandler.Delete(responseRecorder, request)

				Expect(responseRecorder.Code).To(Equal(http.StatusUnauthorized))
			})
		})
	})

	Describe(".Upsert", func() {
		Context("POST", func() {
			var (
				route []db.Route
			)

			BeforeEach(func() {
				route = []db.Route{
					{
						Route: "post_here",
						IP:    "1.2.3.4",
						Port:  7000,
						TTL:   50,
					},
				}
			})

			Context("when all inputs are present and correct", func() {
				It("returns an http status created", func() {
					request = newTestRequest(route)
					routesHandler.Upsert(responseRecorder, request)

					Expect(responseRecorder.Code).To(Equal(http.StatusCreated))
				})

				It("accepts a list of routes in the body", func() {
					route = append(route, route[0])
					route[1].IP = "5.4.3.2"

					request = newTestRequest(route)
					routesHandler.Upsert(responseRecorder, request)

					Expect(responseRecorder.Code).To(Equal(http.StatusCreated))
					Expect(database.SaveRouteCallCount()).To(Equal(2))
					Expect(database.SaveRouteArgsForCall(0)).To(Equal(route[0]))
					Expect(database.SaveRouteArgsForCall(1)).To(Equal(route[1]))
				})

				It("logs the route declaration", func() {
					request = newTestRequest(route)
					routesHandler.Upsert(responseRecorder, request)

					data := map[string]interface{}{
						"ip":       "1.2.3.4",
						"log_guid": "",
						"port":     float64(7000),
						"route":    "post_here",
						"ttl":      float64(50),
					}
					log_data := map[string][]interface{}{"route_creation": []interface{}{data}}

					Expect(logger.Logs()[0].Message).To(ContainSubstring("request"))
					Expect(logger.Logs()[0].Data["route_creation"]).To(Equal(log_data["route_creation"]))
				})

				It("does not require log guid on the request", func() {
					route[0].LogGuid = ""

					request = newTestRequest(route)
					routesHandler.Upsert(responseRecorder, request)

					Expect(responseRecorder.Code).To(Equal(http.StatusCreated))
				})

				It("writes to database backend", func() {
					route[0].LogGuid = "my-guid"

					request = newTestRequest(route)
					routesHandler.Upsert(responseRecorder, request)

					Expect(database.SaveRouteCallCount()).To(Equal(1))
					Expect(database.SaveRouteArgsForCall(0)).To(Equal(route[0]))
				})

				Context("when database fails to save", func() {
					BeforeEach(func() {
						database.SaveRouteReturns(errors.New("stuff broke"))
					})

					It("responds with a server error", func() {
						request = newTestRequest(route)
						routesHandler.Upsert(responseRecorder, request)

						Expect(responseRecorder.Code).To(Equal(http.StatusInternalServerError))
						Expect(responseRecorder.Body.String()).To(ContainSubstring("stuff broke"))
					})
				})
			})

			Context("when there are errors with the input", func() {
				BeforeEach(func() {
					validator.ValidateCreateReturns(&handlers.Error{"a type", "error message"})
				})

				It("does not write to the key-value store backend", func() {
					request = newTestRequest(route)
					routesHandler.Upsert(responseRecorder, request)

					Expect(database.SaveRouteCallCount()).To(Equal(0))
				})

				It("logs the error", func() {
					request = newTestRequest(route)
					routesHandler.Upsert(responseRecorder, request)

					Expect(logger.Logs()[1].Message).To(ContainSubstring("error"))
					Expect(logger.Logs()[1].Data["error"]).To(Equal("error message"))
				})
			})

			Context("when the UAA token is not valid", func() {
				BeforeEach(func() {
					token.DecodeTokenReturns(errors.New("Not valid"))
				})

				It("returns an Unauthorized status code", func() {
					request = newTestRequest(route)
					routesHandler.Upsert(responseRecorder, request)

					Expect(responseRecorder.Code).To(Equal(http.StatusUnauthorized))
				})
			})
		})
	})
})
