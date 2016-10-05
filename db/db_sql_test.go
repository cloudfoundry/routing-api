package db_test

import (
	"errors"
	"os"
	"sync/atomic"
	"time"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/routing-api/config"
	"code.cloudfoundry.org/routing-api/db"
	"code.cloudfoundry.org/routing-api/db/fakes"
	"code.cloudfoundry.org/routing-api/matchers"
	"code.cloudfoundry.org/routing-api/migration"
	"code.cloudfoundry.org/routing-api/models"
	"github.com/jinzhu/gorm"
	"github.com/nu7hatch/gouuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("SqlDB", func() {
	var (
		sqlDB *db.SqlDB
		err   error
	)
	BeforeEach(func() {
		sqlCfg = &config.SqlDB{
			Username: "root",
			Password: "password",
			Schema:   sqlDBName,
			Host:     "localhost",
			Port:     3306,
			Type:     "mysql",
		}
		sqlDB, err = db.NewSqlDB(sqlCfg)
		Expect(err).ToNot(HaveOccurred())
		migration.NewV0InitMigration().Run(sqlDB)
	})

	AfterEach(func() {
		_, ok := sqlDB.Client.(*gorm.DB)
		if ok {
			Expect(sqlDB.Client.Delete(&models.Route{}).Error).ToNot(HaveOccurred())
			Expect(sqlDB.Client.Delete(&models.TcpRouteMapping{}).Error).ToNot(HaveOccurred())
			Expect(sqlDB.Client.Delete(&models.RouterGroupDB{}).Error).ToNot(HaveOccurred())
		}
	})

	Describe("Connection", func() {
		var sqlDB db.DB
		JustBeforeEach(func() {
			sqlDB, err = db.NewSqlDB(sqlCfg)
		})

		It("returns a sql db client", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(sqlDB).ToNot(BeNil())
		})

		Context("when config is nil", func() {
			BeforeEach(func() {
				sqlCfg = nil
			})

			It("returns an error", func() {
				Expect(err).To(HaveOccurred())
				Expect(sqlDB).To(BeNil())
			})
		})

		Context("when authentication fails", func() {
			BeforeEach(func() {
				sqlCfg.Password = "wrong_password"
			})

			It("returns an error", func() {
				Expect(err).To(HaveOccurred())
				Expect(sqlDB).To(BeNil())
			})
		})

		Context("when connecting to SQL DB fails", func() {
			BeforeEach(func() {
				sqlCfg.Port = 1234
			})

			It("returns an error", func() {
				Expect(err).To(HaveOccurred())
				Expect(sqlDB).To(BeNil())
			})
		})
	})

	Describe("ReadRouterGroups", func() {
		var (
			routerGroups models.RouterGroups
			err          error
			rg           models.RouterGroupDB
		)

		JustBeforeEach(func() {
			routerGroups, err = sqlDB.ReadRouterGroups()
		})

		Context("when there are router groups", func() {
			BeforeEach(func() {
				rg = models.RouterGroupDB{
					Model:           models.Model{Guid: newUuid()},
					Name:            "rg-1",
					Type:            "tcp",
					ReservablePorts: "120",
				}
				Expect(sqlDB.Client.Create(&rg).Error).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				Expect(sqlDB.Client.Delete(&rg).Error).ToNot(HaveOccurred())
			})

			It("returns list of router groups", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(routerGroups).ToNot(BeNil())
				Expect(routerGroups).To(HaveLen(1))
				Expect(routerGroups[0]).Should(matchers.MatchRouterGroup(rg.ToRouterGroup()))
			})
		})

		Context("when there are no router groups", func() {
			BeforeEach(func() {
				sqlDB.Client.Where("1 = 1").Delete(&models.RouterGroupDB{})
			})

			It("returns an empty slice", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(routerGroups).ToNot(BeNil())
				Expect(routerGroups).To(HaveLen(0))
			})
		})

		Context("when there is a connection error", func() {
			BeforeEach(func() {
				fakeClient := &fakes.FakeClient{}
				fakeClient.FindReturns(&gorm.DB{Error: errors.New("connection refused")})
				sqlDB.Client = fakeClient
			})

			It("returns an error", func() {
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("ReadRouterGroup", func() {
		var (
			routerGroup   models.RouterGroup
			err           error
			rg            models.RouterGroupDB
			routerGroupId string
		)

		JustBeforeEach(func() {
			routerGroup, err = sqlDB.ReadRouterGroup(routerGroupId)
		})

		Context("when router group exists", func() {
			BeforeEach(func() {
				routerGroupId = newUuid()
				rg = models.RouterGroupDB{
					Model:           models.Model{Guid: routerGroupId},
					Name:            "rg-1",
					Type:            "tcp",
					ReservablePorts: "120",
				}
				Expect(sqlDB.Client.Create(&rg).Error).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				Expect(sqlDB.Client.Delete(&rg).Error).ToNot(HaveOccurred())
			})

			It("returns the router group", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(routerGroup.Guid).To(Equal(rg.Guid))
				Expect(routerGroup.Name).To(Equal(rg.Name))
				Expect(string(routerGroup.ReservablePorts)).To(Equal(rg.ReservablePorts))
				Expect(string(routerGroup.Type)).To(Equal(rg.Type))
			})
		})

		Context("when router group doesn't exist", func() {
			BeforeEach(func() {
				routerGroupId = newUuid()
			})

			It("returns an empty struct", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(routerGroup).To(Equal(models.RouterGroup{}))
			})
		})
	})

	Describe("SaveRouterGroup", func() {
		var (
			routerGroup   models.RouterGroup
			err           error
			routerGroupId string
		)
		BeforeEach(func() {
			routerGroupId = newUuid()
			routerGroup = models.RouterGroup{
				Guid:            routerGroupId,
				Name:            "router-group-1",
				Type:            "tcp",
				ReservablePorts: "65000-65002",
			}
		})

		JustBeforeEach(func() {
			err = sqlDB.SaveRouterGroup(routerGroup)
		})

		Context("when the router group already exists", func() {
			BeforeEach(func() {
				sqlDB.Client.Create(&models.RouterGroupDB{
					Model:           models.Model{Guid: routerGroupId},
					Name:            "rg-1",
					Type:            "tcp",
					ReservablePorts: "120",
				})
			})

			AfterEach(func() {
				sqlDB.Client.Delete(&models.RouterGroupDB{
					Model: models.Model{Guid: routerGroupId},
				})
			})

			It("updates the existing router group", func() {
				Expect(err).ToNot(HaveOccurred())
				rg, err := sqlDB.ReadRouterGroup(routerGroup.Guid)
				Expect(err).ToNot(HaveOccurred())

				Expect(rg.Guid).To(Equal(routerGroup.Guid))
				Expect(rg.Name).To(Equal(routerGroup.Name))
				Expect(rg.ReservablePorts).To(Equal(routerGroup.ReservablePorts))
				Expect(rg.Type).To(Equal(routerGroup.Type))
			})
		})

		Context("when router group doesn't exist", func() {
			It("creates the router group", func() {
				Expect(err).ToNot(HaveOccurred())
				rg, err := sqlDB.ReadRouterGroup(routerGroup.Guid)
				Expect(err).ToNot(HaveOccurred())
				Expect(rg.Guid).To(Equal(routerGroup.Guid))
				Expect(rg.Name).To(Equal(routerGroup.Name))
				Expect(rg.ReservablePorts).To(Equal(routerGroup.ReservablePorts))
				Expect(rg.Type).To(Equal(routerGroup.Type))
			})
		})
	})

	Describe("SaveTcpRouteMapping", func() {
		var (
			routerGroupId string
			tcpRoute      models.TcpRouteMapping
		)

		BeforeEach(func() {
			routerGroupId = newUuid()
			tcpRoute = models.NewTcpRouteMapping(routerGroupId, 3056, "127.0.0.1", 2990, 5)
		})

		AfterEach(func() {
			sqlDB.Client.Delete(&tcpRoute)
		})

		Context("when tcp route exists", func() {
			BeforeEach(func() {
				err = sqlDB.SaveTcpRouteMapping(tcpRoute)
				Expect(err).ToNot(HaveOccurred())
			})

			It("updates the existing tcp route mapping and increments modification tag", func() {
				err := sqlDB.SaveTcpRouteMapping(tcpRoute)
				Expect(err).ToNot(HaveOccurred())
				var dbTcpRoute models.TcpRouteMapping
				sqlDB.Client.Where("host_ip = ?", "127.0.0.1").First(&dbTcpRoute)
				Expect(dbTcpRoute).ToNot(BeNil())
				Expect(dbTcpRoute.ModificationTag.Index).To(BeNumerically("==", 1))
			})

			It("refreshes the expiration time of the mapping", func() {
				var dbTcpRoute models.TcpRouteMapping
				var ttl = 9
				sqlDB.Client.Where("host_ip = ?", "127.0.0.1").First(&dbTcpRoute)
				Expect(dbTcpRoute).ToNot(BeNil())
				initialExpiration := dbTcpRoute.ExpiresAt

				tcpRoute.TTL = &ttl
				err := sqlDB.SaveTcpRouteMapping(tcpRoute)
				Expect(err).ToNot(HaveOccurred())

				sqlDB.Client.Where("host_ip = ?", "127.0.0.1").First(&dbTcpRoute)
				Expect(dbTcpRoute).ToNot(BeNil())
				Expect(initialExpiration).To(BeTemporally("<", dbTcpRoute.ExpiresAt))
			})
		})

		Context("when the tcp route doesn't exist", func() {
			It("creates a modification tag", func() {
				err := sqlDB.SaveTcpRouteMapping(tcpRoute)
				Expect(err).ToNot(HaveOccurred())
				var dbTcpRoute models.TcpRouteMapping
				err = sqlDB.Client.Where("host_ip = ?", "127.0.0.1").First(&dbTcpRoute).Error
				Expect(err).ToNot(HaveOccurred())
				Expect(dbTcpRoute.ModificationTag.Guid).ToNot(BeEmpty())
				Expect(dbTcpRoute.ModificationTag.Index).To(BeZero())
			})

			It("creates a tcp route", func() {
				err := sqlDB.SaveTcpRouteMapping(tcpRoute)
				Expect(err).ToNot(HaveOccurred())
				var dbTcpRoute models.TcpRouteMapping
				err = sqlDB.Client.Where("host_ip = ?", "127.0.0.1").First(&dbTcpRoute).Error
				Expect(err).ToNot(HaveOccurred())
				Expect(dbTcpRoute).To(matchers.MatchTcpRoute(tcpRoute))
			})
		})
	})

	Describe("ReadTcpRouteMappings", func() {
		var (
			err       error
			tcpRoutes []models.TcpRouteMapping
		)

		JustBeforeEach(func() {
			tcpRoutes, err = sqlDB.ReadTcpRouteMappings()
		})

		Context("when at least one tcp route exists", func() {
			var (
				routerGroupId     string
				tcpRoute          models.TcpRouteMapping
				tcpRouteWithModel models.TcpRouteMapping
			)

			BeforeEach(func() {
				routerGroupId = newUuid()
				modTag := models.ModificationTag{Guid: "some-tag", Index: 10}
				tcpRoute = models.NewTcpRouteMapping(routerGroupId, 3056, "127.0.0.1", 2990, 5)
				tcpRoute.ModificationTag = modTag
				tcpRouteWithModel, err = models.NewTcpRouteMappingWithModel(tcpRoute)
				Expect(err).NotTo(HaveOccurred())
				Expect(sqlDB.Client.Create(&tcpRouteWithModel).Error).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				Expect(sqlDB.Client.Delete(&tcpRouteWithModel).Error).ToNot(HaveOccurred())
			})

			It("returns the tcp routes", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(tcpRoutes).To(HaveLen(1))
				Expect(tcpRoutes[0].TcpMappingEntity).To(Equal(tcpRoute.TcpMappingEntity))
			})

			Context("when tcp routes have outlived their ttl", func() {
				var (
					routerGroupId            string
					expiredTcpRoute          models.TcpRouteMapping
					expiredTcpRouteWithModel models.TcpRouteMapping
				)

				BeforeEach(func() {
					modTag := models.ModificationTag{Guid: "some-tag", Index: 10}
					expiredTcpRoute = models.NewTcpRouteMapping(routerGroupId, 3057, "127.0.0.1", 2990, -9)
					expiredTcpRoute.ModificationTag = modTag
					expiredTcpRouteWithModel, err = models.NewTcpRouteMappingWithModel(expiredTcpRoute)
					Expect(err).NotTo(HaveOccurred())
					Expect(sqlDB.Client.Create(&expiredTcpRouteWithModel).Error).ToNot(HaveOccurred())
				})

				AfterEach(func() {
					Expect(sqlDB.Client.Delete(&expiredTcpRouteWithModel).Error).ToNot(HaveOccurred())
				})

				It("does not return the tcp route", func() {
					Expect(err).ToNot(HaveOccurred())

					var tcpDBRoutes []models.TcpRouteMapping
					err := sqlDB.Client.Find(&tcpDBRoutes).Error
					Expect(err).NotTo(HaveOccurred())
					Expect(tcpDBRoutes).To(HaveLen(2))

					Expect(tcpRoutes).To(HaveLen(1))
					Expect(tcpRoutes[0].TcpMappingEntity).To(Equal(tcpRoute.TcpMappingEntity))
				})
			})
		})

		Context("when tcp route doesn't exist", func() {
			It("returns an empty array", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(tcpRoutes).To(Equal([]models.TcpRouteMapping{}))
			})
		})
	})

	Describe("DeleteTcpRouteMapping", func() {
		var (
			err               error
			routerGroupId     string
			tcpRoute          models.TcpRouteMapping
			tcpRouteWithModel models.TcpRouteMapping
		)
		BeforeEach(func() {
			routerGroupId = newUuid()
			modTag := models.ModificationTag{Guid: "some-tag", Index: 10}
			tcpRoute = models.NewTcpRouteMapping(routerGroupId, 3056, "127.0.0.1", 2990, 5)
			tcpRoute.ModificationTag = modTag
			tcpRouteWithModel, err = models.NewTcpRouteMappingWithModel(tcpRoute)
			Expect(err).ToNot(HaveOccurred())
		})

		JustBeforeEach(func() {
			err = sqlDB.DeleteTcpRouteMapping(tcpRoute)
		})

		Context("when at least one tcp route exists", func() {
			BeforeEach(func() {
				Expect(sqlDB.Client.Create(&tcpRouteWithModel).Error).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				Expect(sqlDB.Client.Delete(&tcpRouteWithModel).Error).ToNot(HaveOccurred())
			})

			It("deletes the tcp route", func() {
				Expect(err).ToNot(HaveOccurred())

				tcpRoutes, err := sqlDB.ReadTcpRouteMappings()
				Expect(err).ToNot(HaveOccurred())
				Expect(tcpRoutes).ToNot(ContainElement(tcpRoute))
			})

			Context("when multiple tcp routes exist", func() {
				var tcpRouteWithModel2 models.TcpRouteMapping

				BeforeEach(func() {
					modTag := models.ModificationTag{Guid: "some-tag", Index: 10}
					tcpRoute2 := models.NewTcpRouteMapping(routerGroupId, 3057, "127.0.0.1", 2990, 5)
					tcpRoute2.ModificationTag = modTag
					tcpRouteWithModel2, err = models.NewTcpRouteMappingWithModel(tcpRoute2)
					Expect(err).ToNot(HaveOccurred())
					Expect(sqlDB.Client.Create(&tcpRouteWithModel2).Error).ToNot(HaveOccurred())
				})

				AfterEach(func() {
					Expect(sqlDB.Client.Delete(&tcpRouteWithModel2).Error).ToNot(HaveOccurred())
				})

				It("does not delete everything", func() {
					Expect(err).ToNot(HaveOccurred())

					tcpRoutes, err := sqlDB.ReadTcpRouteMappings()
					Expect(err).ToNot(HaveOccurred())

					Expect(tcpRoutes).To(HaveLen(1))
					Expect(tcpRoutes[0]).To(matchers.MatchTcpRoute(tcpRouteWithModel2))
				})
			})
		})

		Context("when the tcp route doesn't exist", func() {
			It("returns an error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).Should(MatchError(db.DeleteError))
			})
		})
	})

	Describe("SaveRoute", func() {
		var (
			httpRoute models.Route
		)

		BeforeEach(func() {
			httpRoute = models.NewRoute("post_here", 7000, "127.0.0.1", "my-guid", "https://rs.com", 5)
		})

		AfterEach(func() {
			sqlDB.Client.Delete(&httpRoute)
		})

		Context("when the http route already exists", func() {
			BeforeEach(func() {
				err = sqlDB.SaveRoute(httpRoute)
				Expect(err).ToNot(HaveOccurred())
			})

			It("updates the existing route and increments its modification tag", func() {
				err := sqlDB.SaveRoute(httpRoute)
				Expect(err).ToNot(HaveOccurred())
				var dbRoute models.Route
				sqlDB.Client.Where("ip = ?", "127.0.0.1").First(&dbRoute)
				Expect(dbRoute).ToNot(BeNil())
				Expect(dbRoute.ModificationTag.Index).To(BeNumerically("==", 1))
			})

			It("refreshes the expiration time of the route", func() {
				var dbRoute models.Route
				var ttl = 9
				sqlDB.Client.Where("ip = ?", "127.0.0.1").First(&dbRoute)
				Expect(dbRoute).ToNot(BeNil())
				initialExpiration := dbRoute.ExpiresAt

				httpRoute.TTL = &ttl
				err := sqlDB.SaveRoute(httpRoute)
				Expect(err).ToNot(HaveOccurred())

				sqlDB.Client.Where("ip = ?", "127.0.0.1").First(&dbRoute)
				Expect(dbRoute).ToNot(BeNil())
				Expect(initialExpiration).To(BeTemporally("<", dbRoute.ExpiresAt))
			})
		})

		Context("when the http route doesn't exist", func() {
			It("creates a modification tag", func() {
				err := sqlDB.SaveRoute(httpRoute)

				Expect(err).ToNot(HaveOccurred())
				var dbRoute models.Route
				err = sqlDB.Client.Where("ip = ?", "127.0.0.1").First(&dbRoute).Error
				Expect(err).ToNot(HaveOccurred())
				Expect(dbRoute.ModificationTag.Guid).ToNot(BeEmpty())
				Expect(dbRoute.ModificationTag.Index).To(BeZero())
			})

			It("creates a http route", func() {
				err := sqlDB.SaveRoute(httpRoute)
				Expect(err).ToNot(HaveOccurred())
				var dbRoute models.Route
				err = sqlDB.Client.Where("ip = ?", "127.0.0.1").First(&dbRoute).Error
				Expect(err).ToNot(HaveOccurred())
				Expect(dbRoute).To(matchers.MatchHttpRoute(httpRoute))
			})
		})
	})

	Describe("ReadRoute", func() {
		var (
			err    error
			routes []models.Route
		)

		JustBeforeEach(func() {
			routes, err = sqlDB.ReadRoutes()
		})

		Context("when at least one route exists", func() {
			var (
				route          models.Route
				routeWithModel models.Route
			)

			BeforeEach(func() {
				modTag := models.ModificationTag{Guid: "some-tag", Index: 10}
				route = models.NewRoute("post_here", 7000, "127.0.0.1", "my-guid", "https://rs.com", 5)
				route.ModificationTag = modTag
				routeWithModel, err = models.NewRouteWithModel(route)
				Expect(err).NotTo(HaveOccurred())
				Expect(sqlDB.Client.Create(&routeWithModel).Error).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				Expect(sqlDB.Client.Delete(&routeWithModel).Error).ToNot(HaveOccurred())
			})

			It("returns the routes", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(routes).To(HaveLen(1))
				Expect(routes[0]).To(matchers.MatchHttpRoute(routeWithModel))
			})

			Context("when http routes have outlived their ttl", func() {
				var (
					expiredRoute          models.Route
					expiredRouteWithModel models.Route
				)

				BeforeEach(func() {
					modTag := models.ModificationTag{Guid: "some-tag", Index: 10}
					expiredRoute = models.NewRoute("post_here", 7001, "127.0.0.1", "my-guid", "https://rs.com", -9)
					expiredRoute.ModificationTag = modTag
					expiredRouteWithModel, err = models.NewRouteWithModel(expiredRoute)
					Expect(err).NotTo(HaveOccurred())
					Expect(sqlDB.Client.Create(&expiredRouteWithModel).Error).ToNot(HaveOccurred())
				})

				AfterEach(func() {
					Expect(sqlDB.Client.Delete(&expiredRouteWithModel).Error).ToNot(HaveOccurred())
				})

				It("does not return the route", func() {
					Expect(err).ToNot(HaveOccurred())

					var dbRoutes []models.Route
					err := sqlDB.Client.Find(&dbRoutes).Error
					Expect(err).NotTo(HaveOccurred())
					Expect(dbRoutes).To(HaveLen(2))

					Expect(routes).To(HaveLen(1))
					Expect(routes[0]).To(matchers.MatchHttpRoute(route))
				})
			})
		})

		Context("when the http route doesn't exist", func() {
			It("returns an empty array", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(routes).To(Equal([]models.Route{}))
			})
		})
	})

	Describe("DeleteRoute", func() {
		var (
			err            error
			route          models.Route
			routeWithModel models.Route
		)
		BeforeEach(func() {
			modTag := models.ModificationTag{Guid: "some-tag", Index: 10}
			route = models.NewRoute("post_here", 7000, "127.0.0.1", "my-guid", "https://rs.com", 100)
			route.ModificationTag = modTag
			routeWithModel, err = models.NewRouteWithModel(route)
			Expect(err).ToNot(HaveOccurred())
		})

		JustBeforeEach(func() {
			err = sqlDB.DeleteRoute(route)
		})

		Context("when at least one route exists", func() {
			BeforeEach(func() {
				Expect(sqlDB.Client.Create(&routeWithModel).Error).ToNot(HaveOccurred())
				routes, err := sqlDB.ReadRoutes()
				Expect(err).ToNot(HaveOccurred())
				Expect(routes).To(HaveLen(1))
				Expect(routes).ToNot(ContainElement(route))
			})

			AfterEach(func() {
				Expect(sqlDB.Client.Delete(&routeWithModel).Error).ToNot(HaveOccurred())
			})

			It("deletes the route", func() {
				Expect(err).ToNot(HaveOccurred())

				routes, err := sqlDB.ReadRoutes()
				Expect(err).ToNot(HaveOccurred())
				Expect(routes).To(BeEmpty())
			})

			Context("when multiple routes exist", func() {
				var (
					routeWithModel2 models.Route
				)
				BeforeEach(func() {
					modTag := models.ModificationTag{Guid: "some-tag", Index: 10}
					route := models.NewRoute("post_here", 7001, "127.0.0.1", "my-guid", "https://rs.com", 5)
					route.ModificationTag = modTag
					routeWithModel2, err = models.NewRouteWithModel(route)
					Expect(err).ToNot(HaveOccurred())
					Expect(sqlDB.Client.Create(&routeWithModel2).Error).ToNot(HaveOccurred())
				})

				AfterEach(func() {
					Expect(sqlDB.Client.Delete(&routeWithModel2).Error).ToNot(HaveOccurred())
				})

				It("deletes the specified route", func() {
					Expect(err).ToNot(HaveOccurred())

					routes, err := sqlDB.ReadRoutes()
					Expect(err).ToNot(HaveOccurred())
					Expect(routes).To(HaveLen(1))
					Expect(routes[0]).To(matchers.MatchHttpRoute(routeWithModel2))
				})
			})
		})

		Context("when the route doesn't exist", func() {
			It("returns an error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).Should(MatchError(db.DeleteError))
			})
		})
	})

	Describe("WatchChanges with tcp events", func() {
		var (
			routerGroupId string
		)

		BeforeEach(func() {
			routerGroupId = newUuid()
		})

		It("does not return an error when canceled", func() {
			_, errors, cancel := sqlDB.WatchChanges(db.TCP_WATCH)
			cancel()
			Consistently(errors).ShouldNot(Receive())
			Eventually(errors).Should(BeClosed())
		})

		Context("with wrong event type", func() {
			It("throws an error", func() {
				event, err, _ := sqlDB.WatchChanges("some-random-key")
				Eventually(err).Should(Receive())
				Eventually(err).Should(BeClosed())

				Consistently(event).ShouldNot(Receive())
				Eventually(event).Should(BeClosed())
			})
		})

		Context("when a tcp route is updated", func() {
			var (
				tcpRoute models.TcpRouteMapping
			)

			BeforeEach(func() {
				tcpRoute = models.NewTcpRouteMapping(routerGroupId, 3057, "127.0.0.1", 2990, 50)
				err = sqlDB.SaveTcpRouteMapping(tcpRoute)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return an update watch event", func() {
				results, _, _ := sqlDB.WatchChanges(db.TCP_WATCH)

				err = sqlDB.SaveTcpRouteMapping(tcpRoute)
				Expect(err).NotTo(HaveOccurred())

				var event db.Event
				Eventually(results).Should((Receive(&event)))
				Expect(event).NotTo(BeNil())
				Expect(event.Type).To(Equal(db.UpdateEvent))
				Expect(event.Value).To(ContainSubstring(`"port":3057`))
			})
		})

		Context("when a tcp route is created", func() {
			It("should return an create watch event", func() {
				results, _, _ := sqlDB.WatchChanges(db.TCP_WATCH)

				tcpRoute := models.NewTcpRouteMapping(routerGroupId, 3057, "127.0.0.1", 2990, 50)
				err = sqlDB.SaveTcpRouteMapping(tcpRoute)
				Expect(err).NotTo(HaveOccurred())

				var event db.Event
				Eventually(results).Should((Receive(&event)))
				Expect(event).NotTo(BeNil())
				Expect(event.Type).To(Equal(db.CreateEvent))
				Expect(event.Value).To(ContainSubstring(`"port":3057`))
			})
		})

		Context("when a route is deleted", func() {
			It("should return an delete watch event", func() {
				tcpRoute := models.NewTcpRouteMapping(routerGroupId, 3057, "127.0.0.1", 2990, 50)
				err := sqlDB.SaveTcpRouteMapping(tcpRoute)
				Expect(err).NotTo(HaveOccurred())

				results, _, _ := sqlDB.WatchChanges(db.TCP_WATCH)

				err = sqlDB.DeleteTcpRouteMapping(tcpRoute)
				Expect(err).NotTo(HaveOccurred())

				var event db.Event
				Eventually(results).Should((Receive(&event)))
				Expect(event).NotTo(BeNil())
				Expect(event.Type).To(Equal(db.DeleteEvent))
				Expect(event.Value).To(ContainSubstring(`"port":3057`))
			})
		})

		Context("Cancel Watches", func() {
			It("cancels any in-flight watches", func() {
				results, err, _ := sqlDB.WatchChanges(db.TCP_WATCH)
				results2, err2, _ := sqlDB.WatchChanges(db.TCP_WATCH)

				sqlDB.CancelWatches()

				Eventually(err).Should(BeClosed())
				Eventually(results).Should(BeClosed())
				Eventually(err2).Should(BeClosed())
				Eventually(results2).Should(BeClosed())
			})

			It("doesn't panic when called twice", func() {
				sqlDB.CancelWatches()
				sqlDB.CancelWatches()
			})

			It("causes subsequent calls to WatchChanges to error", func() {
				sqlDB.CancelWatches()

				event, err, _ := sqlDB.WatchChanges(db.TCP_WATCH)
				Eventually(err).ShouldNot(Receive())
				Eventually(err).Should(BeClosed())

				Consistently(event).ShouldNot(Receive())
				Eventually(event).Should(BeClosed())

			})
		})
	})

	Describe("WatchChanges with http events", func() {
		It("does not return an error when canceled", func() {
			_, errors, cancel := sqlDB.WatchChanges(db.HTTP_WATCH)
			cancel()
			Consistently(errors).ShouldNot(Receive())
			Eventually(errors).Should(BeClosed())
		})

		Context("when a http route is updated", func() {
			var (
				httpRoute models.Route
			)

			BeforeEach(func() {
				httpRoute = models.NewRoute("post_here", 7001, "127.0.0.1", "my-guid", "https://rs.com", 5)
				err = sqlDB.SaveRoute(httpRoute)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return an update watch event", func() {
				results, _, _ := sqlDB.WatchChanges(db.HTTP_WATCH)

				err = sqlDB.SaveRoute(httpRoute)
				Expect(err).NotTo(HaveOccurred())

				var event db.Event
				Eventually(results).Should((Receive(&event)))
				Expect(event).NotTo(BeNil())
				Expect(event.Type).To(Equal(db.UpdateEvent))
				Expect(event.Value).To(ContainSubstring(`"port":7001`))
			})
		})

		Context("when a http route is created", func() {
			It("should return an create watch event", func() {
				results, _, _ := sqlDB.WatchChanges(db.HTTP_WATCH)

				httpRoute := models.NewRoute("post_here", 7002, "127.0.0.1", "my-guid", "https://rs.com", 5)
				err := sqlDB.SaveRoute(httpRoute)
				Expect(err).NotTo(HaveOccurred())

				var event db.Event
				Eventually(results).Should((Receive(&event)))
				Expect(event).NotTo(BeNil())
				Expect(event.Type).To(Equal(db.CreateEvent))
				Expect(event.Value).To(ContainSubstring(`"port":7002`))
			})
		})

		Context("when a http route is deleted", func() {
			It("should return an delete watch event", func() {
				httpRoute := models.NewRoute("post_here", 7003, "127.0.0.1", "my-guid", "https://rs.com", 5)
				err := sqlDB.SaveRoute(httpRoute)
				Expect(err).NotTo(HaveOccurred())

				results, _, _ := sqlDB.WatchChanges(db.HTTP_WATCH)

				err = sqlDB.DeleteRoute(httpRoute)
				Expect(err).NotTo(HaveOccurred())

				var event db.Event
				Eventually(results).Should((Receive(&event)))
				Expect(event).NotTo(BeNil())
				Expect(event.Type).To(Equal(db.DeleteEvent))
				Expect(event.Value).To(ContainSubstring(`"port":7003`))
			})
		})

		Context("Cancel Watches", func() {
			It("cancels any in-flight watches", func() {
				results, err, _ := sqlDB.WatchChanges(db.HTTP_WATCH)
				results2, err2, _ := sqlDB.WatchChanges(db.HTTP_WATCH)

				sqlDB.CancelWatches()

				Eventually(err).Should(BeClosed())
				Eventually(results).Should(BeClosed())
				Eventually(err2).Should(BeClosed())
				Eventually(results2).Should(BeClosed())
			})

			It("causes subsequent calls to WatchChanges to error", func() {
				sqlDB.CancelWatches()

				event, err, _ := sqlDB.WatchChanges(db.HTTP_WATCH)
				Eventually(err).ShouldNot(Receive())
				Eventually(err).Should(BeClosed())

				Consistently(event).ShouldNot(Receive())
				Eventually(event).Should(BeClosed())

			})
		})
	})

	Describe("Cleanup routes", func() {
		var (
			logger  lager.Logger
			signals chan os.Signal
		)

		BeforeEach(func() {
			signals = make(chan os.Signal, 1)
		})

		JustBeforeEach(func() {
			logger = lagertest.NewTestLogger("prune")
			go sqlDB.CleanupRoutes(logger, 100*time.Millisecond, signals)
		})

		AfterEach(func() {
			close(signals)
		})

		Context("when cleanup takes longer than the cleanup interval", func() {
			var (
				fakeClient *fakes.FakeClient
				done       chan bool
				count      int32
			)

			BeforeEach(func() {
				done = make(chan bool, 2)
				fakeClient = &fakes.FakeClient{}
				fakeClient.DeleteStub = func(value interface{}, where ...interface{}) *gorm.DB {
					time.Sleep(500 * time.Millisecond)
					c := atomic.AddInt32(&count, 1)
					if c <= 2 {
						done <- true
					}
					return &gorm.DB{}
				}
				fakeClient.FindReturns(&gorm.DB{})
				sqlDB.Client = fakeClient
			})

			AfterEach(func() {
				close(done)
			})

			It("should not cleanup before the previous cleanup is complete", func() {
				Eventually(fakeClient.DeleteCallCount).Should(Equal(2))
				Eventually(done).Should(Receive())
				Eventually(done).Should(Receive())

				Eventually(fakeClient.DeleteCallCount).Should(Equal(4))
			})

		})

		Context("tcp routes", func() {
			var tcpRouteModel models.TcpRouteMapping

			BeforeEach(func() {
				tcpRoute := models.NewTcpRouteMapping("guid", 3555, "127.0.0.1", 7879, 1)
				var err error
				tcpRouteModel, err = models.NewTcpRouteMappingWithModel(tcpRoute)
				Expect(err).ToNot(HaveOccurred())
				sqlDB.SaveTcpRouteMapping(tcpRouteModel)

				routes, err := sqlDB.ReadTcpRouteMappings()
				Expect(routes).To(HaveLen(1))
			})

			Context("when db connection is successful", func() {
				Context("when all routes have expired", func() {

					It("should prune the expired routes and log the number of pruned routes", func() {
						Eventually(func() []models.TcpRouteMapping {
							var tcpRoutes []models.TcpRouteMapping
							err := sqlDB.Client.Where("host_ip = ?", "127.0.0.1").Find(&tcpRoutes).Error
							Expect(err).ToNot(HaveOccurred())
							return tcpRoutes
						}, 3).Should(HaveLen(0))
						Eventually(logger, 2).Should(gbytes.Say(`"prune.successfully-finished-pruning-tcp-routes","log_level":1,"data":{"rowsAffected":1}`))
					})

					It("should emit a ExpireEvent for the pruned route", func() {
						results, _, _ := sqlDB.WatchChanges(db.TCP_WATCH)
						var event db.Event
						Eventually(results, 3).Should((Receive(&event)))
						Expect(event).NotTo(BeNil())
						Expect(event.Type).To(Equal(db.ExpireEvent))
						Expect(event.Value).To(ContainSubstring(`"port":3555`))
					})
				})

				Context("when routes that have not expired exist", func() {
					var tcpRoute models.TcpRouteMapping

					BeforeEach(func() {
						tcpRoute = models.NewTcpRouteMapping("guid", 3556, "127.0.0.1", 7879, 100)
						sqlDB.SaveTcpRouteMapping(tcpRoute)

						var routesDB []models.TcpRouteMapping
						sqlDB.Client.Find(&routesDB)
						Expect(routesDB).To(HaveLen(2))
					})

					It("should only prune expired routes", func() {
						var tcpRoutes []models.TcpRouteMapping

						Eventually(func() []models.TcpRouteMapping {
							err := sqlDB.Client.Where("host_ip = ?", "127.0.0.1").Find(&tcpRoutes).Error
							Expect(err).ToNot(HaveOccurred())
							return tcpRoutes
						}, 2).Should(HaveLen(1))

						Expect(tcpRoutes[0]).To(matchers.MatchTcpRoute(tcpRoute))
					})
				})
			})

		})

		Context("http routes", func() {
			BeforeEach(func() {
				httpRoute := models.NewRoute("post_here", 7000, "127.0.0.1", "my-guid", "https://rs.com", 1)
				httpRouteModel, err := models.NewRouteWithModel(httpRoute)
				Expect(err).ToNot(HaveOccurred())
				err = sqlDB.SaveRoute(httpRouteModel)
				Expect(err).ToNot(HaveOccurred())

				routes, err := sqlDB.ReadRoutes()
				Expect(err).ToNot(HaveOccurred())
				Expect(routes).To(HaveLen(1))
			})

			Context("when db connection is successful", func() {
				Context("when all routes have expired", func() {

					It("should prune the expired routes and log the number of pruned routes", func() {
						Eventually(func() []models.Route {
							var httpRoutes []models.Route
							err := sqlDB.Client.Where("ip = ?", "127.0.0.1").Find(&httpRoutes).Error
							Expect(err).ToNot(HaveOccurred())
							return httpRoutes
						}, 3).Should(HaveLen(0))

						Eventually(logger, 2).Should(gbytes.Say(`prune.successfully-finished-pruning-http-routes","log_level":1,"data":{"rowsAffected":1}`))
					})

					It("should emit a ExpireEvent for the pruned route", func() {
						results, _, _ := sqlDB.WatchChanges(db.HTTP_WATCH)
						var event db.Event
						Eventually(results, 3).Should((Receive(&event)))
						Expect(event).NotTo(BeNil())
						Expect(event.Type).To(Equal(db.ExpireEvent))
						Expect(event.Value).To(ContainSubstring(`"port":7000`))
					})
				})

				Context("when some routes are expired", func() {
					var httpRoute models.Route

					BeforeEach(func() {
						httpRoute = models.NewRoute("post_here", 7001, "127.0.0.1", "my-guid", "https://rs.com", 100)
						err := sqlDB.SaveRoute(httpRoute)
						Expect(err).ToNot(HaveOccurred())

						var dbRoutes []models.Route
						sqlDB.Client.Where("ip = ?", "127.0.0.1").Find(&dbRoutes)
						Expect(dbRoutes).To(HaveLen(2))
					})

					It("should prune only expired routes", func() {
						var httpRoutes []models.Route

						Eventually(func() []models.Route {
							err := sqlDB.Client.Where("ip = ?", "127.0.0.1").Find(&httpRoutes).Error
							Expect(err).ToNot(HaveOccurred())
							return httpRoutes
						}, 3).Should(HaveLen(1))

						Expect(httpRoutes[0]).To(matchers.MatchHttpRoute(httpRoute))
					})
				})
			})
		})

		Context("when db throws an error", func() {
			BeforeEach(func() {
				sqlDB.Client.Close()
			})

			AfterEach(func() {
				sqlDB, err = db.NewSqlDB(sqlCfg)
				Expect(err).ToNot(HaveOccurred())
			})

			It("logs error message", func() {
				Eventually(logger, 2).Should(gbytes.Say(`failed-to-prune-.*-routes","log_level":2,"data":{"error":"sql: database is closed"}`))
				Eventually(logger, 2).Should(gbytes.Say(`failed-to-prune-.*-routes","log_level":2,"data":{"error":"sql: database is closed"}`))
			})
		})
	})
})

func newUuid() string {
	u, err := uuid.NewV4()
	Expect(err).ToNot(HaveOccurred())
	return u.String()
}
