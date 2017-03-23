package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"

	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/routing-api"
	fake_db "code.cloudfoundry.org/routing-api/db/fakes"
	"code.cloudfoundry.org/routing-api/handlers"
	"code.cloudfoundry.org/routing-api/metrics"
	"code.cloudfoundry.org/routing-api/models"
	fake_client "code.cloudfoundry.org/uaa-go-client/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/rata"
)

const (
	DefaultRouterGroupGuid      = "bad25cff-9332-48a6-8603-b619858e7992"
	DefaultHTTPRouterGroupGuid  = "sad25cff-9332-48a6-8603-b619858e7992"
	DefaultOtherRouterGroupGuid = "mad25cff-9332-48a6-8603-b619858e7992"
	DefaultRouterGroupName      = "default-tcp"
	DefaultRouterGroupType      = "tcp"
)

var _ = Describe("RouterGroupsHandler", func() {

	var (
		routerGroupHandler *handlers.RouterGroupsHandler
		request            *http.Request
		responseRecorder   *httptest.ResponseRecorder
		fakeClient         *fake_client.FakeClient
		fakeDb             *fake_db.FakeDB
		logger             *lagertest.TestLogger
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test-router-group")
		fakeClient = &fake_client.FakeClient{}
		fakeDb = &fake_db.FakeDB{}
		routerGroupHandler = handlers.NewRouteGroupsHandler(fakeClient, logger, fakeDb)
		responseRecorder = httptest.NewRecorder()

		fakeRouterGroups := []models.RouterGroup{
			{
				Guid:            DefaultRouterGroupGuid,
				Name:            DefaultRouterGroupName,
				Type:            DefaultRouterGroupType,
				ReservablePorts: "1024-65535",
			},
		}
		fakeDb.ReadRouterGroupsReturns(fakeRouterGroups, nil)
	})

	Describe("ListRouterGroups", func() {
		It("responds with 200 OK and returns default router group details", func() {
			var err error
			request, err = http.NewRequest("GET", routing_api.ListRouterGroups, nil)
			Expect(err).NotTo(HaveOccurred())
			routerGroupHandler.ListRouterGroups(responseRecorder, request)
			Expect(responseRecorder.Code).To(Equal(http.StatusOK))
			payload := responseRecorder.Body.String()
			Expect(payload).To(MatchJSON(`[
			{
				"guid": "bad25cff-9332-48a6-8603-b619858e7992",
				"name": "default-tcp",
				"type": "tcp",
				"reservable_ports": "1024-65535"
			}]`))
		})

		It("checks for routing.router_groups.read scope", func() {
			var err error
			request, err = http.NewRequest("GET", routing_api.ListRouterGroups, nil)
			Expect(err).NotTo(HaveOccurred())
			routerGroupHandler.ListRouterGroups(responseRecorder, request)
			_, permission := fakeClient.DecodeTokenArgsForCall(0)
			Expect(permission).To(ConsistOf(handlers.RouterGroupsReadScope))
		})

		Context("when the db fails to save router group", func() {
			BeforeEach(func() {
				fakeDb.ReadRouterGroupsReturns([]models.RouterGroup{}, errors.New("db communication failed"))
			})

			It("returns a DB communication error", func() {
				var err error
				request, err = http.NewRequest("GET", routing_api.ListRouterGroups, nil)
				Expect(err).NotTo(HaveOccurred())
				routerGroupHandler.ListRouterGroups(responseRecorder, request)
				Expect(fakeDb.ReadRouterGroupsCallCount()).To(Equal(1))
				Expect(responseRecorder.Code).To(Equal(http.StatusInternalServerError))
				payload := responseRecorder.Body.String()
				Expect(payload).To(MatchJSON(`{
					"name": "DBCommunicationError",
					"message": "db communication failed"
				}`))
			})
		})

		Context("when authorization token is invalid", func() {
			var (
				currentCount int64
			)
			BeforeEach(func() {
				currentCount = metrics.GetTokenErrors()
				fakeClient.DecodeTokenReturns(errors.New("kaboom"))
			})

			It("returns Unauthorized error", func() {
				var err error
				request, err = http.NewRequest("GET", routing_api.ListRouterGroups, nil)
				Expect(err).NotTo(HaveOccurred())
				routerGroupHandler.ListRouterGroups(responseRecorder, request)
				Expect(responseRecorder.Code).To(Equal(http.StatusUnauthorized))
				Expect(metrics.GetTokenErrors()).To(Equal(currentCount + 1))
			})
		})

	})

	Describe("UpdateRouterGroup", func() {
		var (
			existingTCPRouterGroup   models.RouterGroup
			existingHTTPRouterGroup  models.RouterGroup
			existingOtherRouterGroup models.RouterGroup
			handler                  http.Handler
			body                     io.Reader
		)

		BeforeEach(func() {
			var err error
			existingTCPRouterGroup = models.RouterGroup{
				Guid:            DefaultRouterGroupGuid,
				Name:            DefaultRouterGroupName,
				Type:            DefaultRouterGroupType,
				ReservablePorts: "1024-65535",
			}

			existingHTTPRouterGroup = models.RouterGroup{
				Guid:            DefaultHTTPRouterGroupGuid,
				Name:            "default-http",
				Type:            "http",
				ReservablePorts: "",
			}

			existingOtherRouterGroup = models.RouterGroup{
				Guid:            DefaultOtherRouterGroupGuid,
				Name:            "default-other",
				Type:            "other",
				ReservablePorts: "9876",
			}

			fakeDb.ReadRouterGroupStub = func(guid string) (models.RouterGroup, error) {
				if guid == DefaultHTTPRouterGroupGuid {
					return existingHTTPRouterGroup, nil
				}
				if guid == DefaultOtherRouterGroupGuid {
					return existingOtherRouterGroup, nil
				}
				return existingTCPRouterGroup, nil
			}

			routes := rata.Routes{
				routing_api.RoutesMap[routing_api.UpdateRouterGroup],
			}
			handler, err = rata.NewRouter(routes, rata.Handlers{
				routing_api.UpdateRouterGroup: http.HandlerFunc(routerGroupHandler.UpdateRouterGroup),
			})
			Expect(err).NotTo(HaveOccurred())
			queryGroup := models.RouterGroup{
				ReservablePorts: "8000",
			}
			bodyBytes, err := json.Marshal(queryGroup)
			Expect(err).ToNot(HaveOccurred())
			body = bytes.NewReader(bodyBytes)
		})

		It("saves the router group", func() {
			var err error
			request, err = http.NewRequest(
				"PUT",
				fmt.Sprintf("/routing/v1/router_groups/%s", DefaultRouterGroupGuid),
				body,
			)
			Expect(err).NotTo(HaveOccurred())

			handler.ServeHTTP(responseRecorder, request)

			Expect(fakeDb.ReadRouterGroupCallCount()).To(Equal(1))
			guid := fakeDb.ReadRouterGroupArgsForCall(0)
			Expect(guid).To(Equal(DefaultRouterGroupGuid))

			Expect(fakeDb.SaveRouterGroupCallCount()).To(Equal(1))
			savedGroup := fakeDb.SaveRouterGroupArgsForCall(0)
			updatedGroup := models.RouterGroup{
				Guid:            DefaultRouterGroupGuid,
				Name:            DefaultRouterGroupName,
				Type:            DefaultRouterGroupType,
				ReservablePorts: "8000",
			}
			Expect(savedGroup).To(Equal(updatedGroup))

			Expect(responseRecorder.Code).To(Equal(http.StatusOK))
			payload := responseRecorder.Body.String()
			Expect(payload).To(MatchJSON(`
			{
			"guid": "bad25cff-9332-48a6-8603-b619858e7992",
			"name": "default-tcp",
			"type": "tcp",
			"reservable_ports": "8000"
			}`))
		})

		It("adds X-Cf-Warnings header", func() {
			var err error
			request, err = http.NewRequest(
				"PUT",
				fmt.Sprintf("/routing/v1/router_groups/%s", DefaultRouterGroupGuid),
				body,
			)
			Expect(err).NotTo(HaveOccurred())

			handler.ServeHTTP(responseRecorder, request)
			warning := responseRecorder.HeaderMap.Get("X-Cf-Warnings")
			Expect(url.QueryUnescape(warning)).To(ContainSubstring("routes becoming inaccessible"))
		})

		Context("when reservable port field is invalid", func() {
			BeforeEach(func() {
				queryGroup := models.RouterGroup{
					ReservablePorts: "fadfadfasdf",
				}
				bodyBytes, err := json.Marshal(queryGroup)
				Expect(err).ToNot(HaveOccurred())
				body = bytes.NewReader(bodyBytes)
			})

			It("does not save the router group", func() {
				var err error
				request, err = http.NewRequest(
					"PUT",
					fmt.Sprintf("/routing/v1/router_groups/%s", DefaultRouterGroupGuid),
					body,
				)
				Expect(err).NotTo(HaveOccurred())

				handler.ServeHTTP(responseRecorder, request)

				Expect(fakeDb.ReadRouterGroupCallCount()).To(Equal(1))
				guid := fakeDb.ReadRouterGroupArgsForCall(0)
				Expect(guid).To(Equal(DefaultRouterGroupGuid))

				Expect(fakeDb.SaveRouterGroupCallCount()).To(Equal(0))
			})

			It("returns a 400 Bad Request", func() {
				var err error
				request, err = http.NewRequest(
					"PUT",
					fmt.Sprintf("/routing/v1/router_groups/%s", DefaultRouterGroupGuid),
					body,
				)
				Expect(err).NotTo(HaveOccurred())

				handler.ServeHTTP(responseRecorder, request)

				Expect(responseRecorder.Code).To(Equal(http.StatusBadRequest))
				payload := responseRecorder.Body.String()
				Expect(payload).To(MatchJSON(`
				{
					"name": "ProcessRequestError",
					"message": "Cannot process request: Port must be between 1024 and 65535"
				}`))
			})
		})

		Context("when adding reservable ports to type: http", func() {
			BeforeEach(func() {
				queryGroup := models.RouterGroup{
					ReservablePorts: "8001",
				}

				bodyBytes, err := json.Marshal(queryGroup)
				Expect(err).ToNot(HaveOccurred())
				body = bytes.NewReader(bodyBytes)
			})

			It("does not save the router group", func() {
				var err error
				request, err = http.NewRequest(
					"PUT",
					fmt.Sprintf("/routing/v1/router_groups/%s", DefaultHTTPRouterGroupGuid),
					body,
				)
				Expect(err).NotTo(HaveOccurred())

				handler.ServeHTTP(responseRecorder, request)

				Expect(fakeDb.ReadRouterGroupCallCount()).To(Equal(1))
				guid := fakeDb.ReadRouterGroupArgsForCall(0)
				Expect(guid).To(Equal(DefaultHTTPRouterGroupGuid))

				Expect(fakeDb.SaveRouterGroupCallCount()).To(Equal(0))
			})

			It("returns a 400 Bad Request", func() {
				var err error
				request, err = http.NewRequest(
					"PUT",
					fmt.Sprintf("/routing/v1/router_groups/%s", DefaultHTTPRouterGroupGuid),
					body,
				)
				Expect(err).NotTo(HaveOccurred())

				handler.ServeHTTP(responseRecorder, request)

				Expect(responseRecorder.Code).To(Equal(http.StatusBadRequest))
				payload := responseRecorder.Body.String()
				Expect(payload).To(MatchJSON(`
				{
					"name": "ProcessRequestError",
					"message": "Cannot process request: Reservable ports are not supported for router groups of type http"
				}`))
			})
		})

		Context("when changing non-http, non-tcp router groups", func() {
			It("saves the router group when changing the ports", func() {
				queryGroup := models.RouterGroup{
					ReservablePorts: "8001",
				}

				bodyBytes, err := json.Marshal(queryGroup)
				Expect(err).ToNot(HaveOccurred())
				body = bytes.NewReader(bodyBytes)

				request, err = http.NewRequest(
					"PUT",
					fmt.Sprintf("/routing/v1/router_groups/%s", DefaultOtherRouterGroupGuid),
					body,
				)
				Expect(err).NotTo(HaveOccurred())

				handler.ServeHTTP(responseRecorder, request)

				Expect(fakeDb.ReadRouterGroupCallCount()).To(Equal(1))
				guid := fakeDb.ReadRouterGroupArgsForCall(0)
				Expect(guid).To(Equal(DefaultOtherRouterGroupGuid))

				Expect(fakeDb.SaveRouterGroupCallCount()).To(Equal(1))
				savedGroup := fakeDb.SaveRouterGroupArgsForCall(0)
				updatedGroup := models.RouterGroup{
					Guid:            DefaultOtherRouterGroupGuid,
					Name:            "default-other",
					Type:            "other",
					ReservablePorts: "8001",
				}
				Expect(savedGroup).To(Equal(updatedGroup))

				Expect(responseRecorder.Code).To(Equal(http.StatusOK))
				payload := responseRecorder.Body.String()
				Expect(payload).To(MatchJSON(`
			{
			"guid": "mad25cff-9332-48a6-8603-b619858e7992",
			"name": "default-other",
			"type": "other",
			"reservable_ports": "8001"
			}`))
			})

			It("saves the router group when removing the ports", func() {
				queryGroup := models.RouterGroup{
					ReservablePorts: "",
				}

				bodyBytes, err := json.Marshal(queryGroup)
				Expect(err).ToNot(HaveOccurred())
				body = bytes.NewReader(bodyBytes)

				request, err = http.NewRequest(
					"PUT",
					fmt.Sprintf("/routing/v1/router_groups/%s", DefaultOtherRouterGroupGuid),
					body,
				)
				Expect(err).NotTo(HaveOccurred())

				handler.ServeHTTP(responseRecorder, request)

				Expect(fakeDb.ReadRouterGroupCallCount()).To(Equal(1))
				guid := fakeDb.ReadRouterGroupArgsForCall(0)
				Expect(guid).To(Equal(DefaultOtherRouterGroupGuid))

				Expect(fakeDb.SaveRouterGroupCallCount()).To(Equal(1))
				savedGroup := fakeDb.SaveRouterGroupArgsForCall(0)
				updatedGroup := models.RouterGroup{
					Guid:            DefaultOtherRouterGroupGuid,
					Name:            "default-other",
					Type:            "other",
					ReservablePorts: "",
				}
				Expect(savedGroup).To(Equal(updatedGroup))

				Expect(responseRecorder.Code).To(Equal(http.StatusOK))
				payload := responseRecorder.Body.String()
				Expect(payload).To(MatchJSON(`
			{
			"guid": "mad25cff-9332-48a6-8603-b619858e7992",
			"name": "default-other",
			"type": "other",
			"reservable_ports": ""
			}`))
			})

		})

		Context("when reservable port field is the empty string for a TCP router group", func() {
			It("does not save the router group", func() {
				var err error

				queryGroup := models.RouterGroup{}
				bodyBytes, err := json.Marshal(queryGroup)
				Expect(err).ToNot(HaveOccurred())
				body := bytes.NewReader(bodyBytes)
				request, err = http.NewRequest(
					"PUT",
					fmt.Sprintf("/routing/v1/router_groups/%s", DefaultRouterGroupGuid),
					body,
				)
				Expect(err).NotTo(HaveOccurred())
				handler.ServeHTTP(responseRecorder, request)

				Expect(fakeDb.ReadRouterGroupCallCount()).To(Equal(1))
				guid := fakeDb.ReadRouterGroupArgsForCall(0)
				Expect(guid).To(Equal(DefaultRouterGroupGuid))

				Expect(fakeDb.SaveRouterGroupCallCount()).To(Equal(0))

				Expect(responseRecorder.Code).To(Equal(http.StatusBadRequest))
				payload := responseRecorder.Body.String()
				Expect(payload).To(MatchJSON(`
				{
					"name": "ProcessRequestError",
					"message": "Cannot process request: Missing reservable_ports in router group: default-tcp"
				}`))
			})
		})

		Context("when reservable port field is not changed", func() {
			It("does not save the router group", func() {
				var err error

				queryGroup := models.RouterGroup{
					ReservablePorts: "1024-65535",
				}
				bodyBytes, err := json.Marshal(queryGroup)
				Expect(err).ToNot(HaveOccurred())
				body := bytes.NewReader(bodyBytes)
				request, err = http.NewRequest(
					"PUT",
					fmt.Sprintf("/routing/v1/router_groups/%s", DefaultRouterGroupGuid),
					body,
				)
				Expect(err).NotTo(HaveOccurred())
				handler.ServeHTTP(responseRecorder, request)

				Expect(fakeDb.ReadRouterGroupCallCount()).To(Equal(1))
				guid := fakeDb.ReadRouterGroupArgsForCall(0)
				Expect(guid).To(Equal(DefaultRouterGroupGuid))

				Expect(fakeDb.SaveRouterGroupCallCount()).To(Equal(0))

				Expect(responseRecorder.Code).To(Equal(http.StatusOK))
				payload := responseRecorder.Body.String()
				Expect(payload).To(MatchJSON(`
				{
				"guid": "bad25cff-9332-48a6-8603-b619858e7992",
				"name": "default-tcp",
				"type": "tcp",
				"reservable_ports": "1024-65535"
				}`))
			})
		})

		It("checks for routing.router_groups.write scope", func() {
			var err error
			updatedGroup := models.RouterGroup{
				ReservablePorts: "8000",
			}
			bodyBytes, err := json.Marshal(updatedGroup)
			Expect(err).ToNot(HaveOccurred())
			body := bytes.NewReader(bodyBytes)
			request, err = http.NewRequest(
				"PUT",
				fmt.Sprintf("/routing/v1/router_groups/%s", DefaultRouterGroupGuid),
				body,
			)
			Expect(err).NotTo(HaveOccurred())
			handler.ServeHTTP(responseRecorder, request)
			_, permission := fakeClient.DecodeTokenArgsForCall(0)
			Expect(permission).To(ConsistOf(handlers.RouterGroupsWriteScope))
		})

		Context("when the router group does not exist", func() {
			BeforeEach(func() {
				fakeDb.ReadRouterGroupReturns(models.RouterGroup{}, nil)
			})

			It("does not save the router group and returns a not found status", func() {
				var err error

				bodyBytes := []byte("{}")
				body := bytes.NewReader(bodyBytes)
				request, err = http.NewRequest(
					"PUT",
					"/routing/v1/router_groups/not-exist",
					body,
				)
				Expect(err).NotTo(HaveOccurred())
				handler.ServeHTTP(responseRecorder, request)
				Expect(fakeDb.ReadRouterGroupCallCount()).To(Equal(1))
				Expect(fakeDb.SaveRouterGroupCallCount()).To(Equal(0))
				Expect(responseRecorder.Code).To(Equal(http.StatusNotFound))
				payload := responseRecorder.Body.String()
				Expect(payload).To(MatchJSON(`{
					"name": "ResourceNotFoundError",
					"message": "Router Group 'not-exist' does not exist"
				}`))
			})
		})

		Context("when the request body is invalid", func() {
			It("does not save the router group and returns a bad request response", func() {
				var err error

				bodyBytes := []byte("invalid json")
				body := bytes.NewReader(bodyBytes)
				request, err = http.NewRequest(
					"PUT",
					fmt.Sprintf("/routing/v1/router_groups/%s", DefaultRouterGroupGuid),
					body,
				)
				Expect(err).NotTo(HaveOccurred())
				handler.ServeHTTP(responseRecorder, request)
				Expect(fakeDb.ReadRouterGroupCallCount()).To(Equal(0))
				Expect(fakeDb.SaveRouterGroupCallCount()).To(Equal(0))
				Expect(responseRecorder.Code).To(Equal(http.StatusBadRequest))
			})
		})

		Context("when the db fails to read router group", func() {
			BeforeEach(func() {
				fakeDb.ReadRouterGroupReturns(models.RouterGroup{}, errors.New("db communication failed"))
			})

			It("returns a DB communication error", func() {
				var err error

				updatedGroup := models.RouterGroup{
					ReservablePorts: "8000",
				}
				bodyBytes, err := json.Marshal(updatedGroup)
				Expect(err).ToNot(HaveOccurred())
				body := bytes.NewReader(bodyBytes)
				request, err = http.NewRequest(
					"PUT",
					fmt.Sprintf("/routing/v1/router_groups/%s", DefaultRouterGroupGuid),
					body,
				)
				Expect(err).NotTo(HaveOccurred())
				handler.ServeHTTP(responseRecorder, request)
				Expect(fakeDb.ReadRouterGroupCallCount()).To(Equal(1))
				Expect(fakeDb.SaveRouterGroupCallCount()).To(Equal(0))
				Expect(responseRecorder.Code).To(Equal(http.StatusInternalServerError))
				payload := responseRecorder.Body.String()
				Expect(payload).To(MatchJSON(`{
					"name": "DBCommunicationError",
					"message": "db communication failed"
				}`))
			})
		})

		Context("when the db fails to save router group", func() {
			BeforeEach(func() {
				fakeDb.SaveRouterGroupReturns(errors.New("db communication failed"))
			})

			It("returns a DB communication error", func() {
				var err error

				updatedGroup := models.RouterGroup{
					ReservablePorts: "8000",
				}
				bodyBytes, err := json.Marshal(updatedGroup)
				Expect(err).ToNot(HaveOccurred())
				body := bytes.NewReader(bodyBytes)
				request, err = http.NewRequest(
					"PUT",
					fmt.Sprintf("/routing/v1/router_groups/%s", DefaultRouterGroupGuid),
					body,
				)
				Expect(err).NotTo(HaveOccurred())
				handler.ServeHTTP(responseRecorder, request)
				Expect(fakeDb.SaveRouterGroupCallCount()).To(Equal(1))
				Expect(responseRecorder.Code).To(Equal(http.StatusInternalServerError))
				payload := responseRecorder.Body.String()
				Expect(payload).To(MatchJSON(`{
					"name": "DBCommunicationError",
					"message": "db communication failed"
				}`))
			})
		})

		Context("when authorization token is invalid", func() {
			var (
				currentCount int64
			)
			BeforeEach(func() {
				currentCount = metrics.GetTokenErrors()
				fakeClient.DecodeTokenReturns(errors.New("kaboom"))
			})

			It("returns Unauthorized error", func() {
				var err error

				updatedGroup := models.RouterGroup{
					ReservablePorts: "8000",
				}
				bodyBytes, err := json.Marshal(updatedGroup)
				Expect(err).ToNot(HaveOccurred())
				body := bytes.NewReader(bodyBytes)
				request, err = http.NewRequest(
					"PUT",
					fmt.Sprintf("/routing/v1/router_groups/%s", DefaultRouterGroupGuid),
					body,
				)
				Expect(err).NotTo(HaveOccurred())
				handler.ServeHTTP(responseRecorder, request)
				Expect(fakeDb.SaveRouterGroupCallCount()).To(Equal(0))
				Expect(responseRecorder.Code).To(Equal(http.StatusUnauthorized))
				Expect(metrics.GetTokenErrors()).To(Equal(currentCount + 1))
			})
		})
	})
})
