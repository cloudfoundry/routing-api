package migration_test

import (
	"time"

	"code.cloudfoundry.org/routing-api/cmd/routing-api/testrunner"
	"code.cloudfoundry.org/routing-api/db"
	"code.cloudfoundry.org/routing-api/migration"
	v7 "code.cloudfoundry.org/routing-api/migration/v7"
	"code.cloudfoundry.org/routing-api/models"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("V7TCPTLSRoutes", func() {
	var (
		sqlDB       *db.SqlDB
		dbAllocator testrunner.DbAllocator
	)

	BeforeEach(func() {
		dbAllocator = testrunner.NewDbAllocator()
		sqlCfg, err := dbAllocator.Create()
		Expect(err).NotTo(HaveOccurred())

		sqlDB, err = db.NewSqlDB(sqlCfg)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		err := dbAllocator.Delete()
		Expect(err).ToNot(HaveOccurred())
	})

	runTests := func() {
		Context("during migration", func() {
			It("allows the migration to occur", func() {
				v7Migration := migration.NewV7TCPTLSRoutes()
				err := v7Migration.Run(sqlDB)
				Expect(err).ToNot(HaveOccurred())

				routes, err := sqlDB.ReadTcpRouteMappings()
				Expect(err).NotTo(HaveOccurred())
				Expect(routes).To(HaveLen(1))
				Expect(routes[0].TerminateFrontendTLS).To(Equal(false))
				Expect(routes[0].ALPN).To(Equal(""))
			})
		})
		Context("After migration", func() {
			BeforeEach(func() {
				v7Migration := migration.NewV7TCPTLSRoutes()
				err := v7Migration.Run(sqlDB)
				Expect(err).ToNot(HaveOccurred())

				sniHostname1 := "sniHostname1"
				tcpRoute1 := models.TcpRouteMapping{
					Model:     models.Model{Guid: "guid-1"},
					ExpiresAt: time.Now().Add(1 * time.Hour),
					TcpMappingEntity: models.TcpMappingEntity{
						RouterGroupGuid: "test1",
						HostPort:        80,
						HostTLSPort:     443,
						HostIP:          "1.2.3.4",
						InstanceId:      "instanceId1",
						ExternalPort:    80,
						SniHostname:     &sniHostname1,

						ModificationTag:      models.ModificationTag{},
						TTL:                  nil,
						IsolationSegment:     "",
						TerminateFrontendTLS: nil,
						ALPN:                 nil,
					},
				}
				_, err = sqlDB.Client.Create(&tcpRoute1)
				Expect(err).NotTo(HaveOccurred())
			})

			It("allows adding non-tls TCP routes without instance-ids", func() {
				sniHostname2 := "sniHostname2"
				tcpRoute2 := models.TcpRouteMapping{
					Model:     models.Model{Guid: "guid-2"},
					ExpiresAt: time.Now().Add(1 * time.Hour),
					TcpMappingEntity: models.TcpMappingEntity{
						RouterGroupGuid: "test1",
						HostPort:        80,
						HostTLSPort:     444,
						HostIP:          "1.2.3.4",
						InstanceId:      "instanceId2",
						ExternalPort:    80,
						SniHostname:     &sniHostname2,
					},
				}
				_, err := sqlDB.Client.Create(&tcpRoute2)
				Expect(err).NotTo(HaveOccurred())

				routes, err := sqlDB.ReadTcpRouteMappings()
				Expect(err).NotTo(HaveOccurred())
				Expect(routes).To(HaveLen(3))
			})
		})
	}

	Describe("Version", func() {
		It("returns 7 for the version", func() {
			v7Migration := migration.NewV7TCPTLSRoutes()
			Expect(v7Migration.Version()).To(Equal(7))
		})
	})

	Describe("Run", func() {
		Context("when there are existing tables with the old tcp_route model", func() {
			BeforeEach(func() {
				err := sqlDB.Client.AutoMigrate(&v7.RouterGroupDB{}, &v7.TcpRouteMapping{}, &v7.Route{})
				Expect(err).ToNot(HaveOccurred())

				sniHostname1 := "sniHostname1"
				tcpRoute1 := v7.TcpRouteMapping{
					Model:     v7.Model{Guid: "guid-0"},
					ExpiresAt: time.Now().Add(1 * time.Hour),
					TcpMappingEntity: v7.TcpMappingEntity{
						RouterGroupGuid: "test0",
						HostPort:        80,
						HostIP:          "1.2.3.4",
						ExternalPort:    80,
						SniHostname:     &sniHostname1,
					},
				}

				_, err = sqlDB.Client.Create(&tcpRoute1)
				Expect(err).NotTo(HaveOccurred())

				v7Migration := migration.NewV7TCPTLSRoutes()
				err = v7Migration.Run(sqlDB)
				Expect(err).ToNot(HaveOccurred())
			})
			runTests()
		})

		Context("when the tables are newly created (by V0 init migration)", func() {
			BeforeEach(func() {
				v0Migration := migration.NewV0InitMigration()
				err := v0Migration.Run(sqlDB)
				Expect(err).ToNot(HaveOccurred())

				sniHostname1 := "sniHostname1"
				tcpRoute1 := v7.TcpRouteMapping{
					Model:     v7.Model{Guid: "guid-0"},
					ExpiresAt: time.Now().Add(1 * time.Hour),
					TcpMappingEntity: v7.TcpMappingEntity{
						RouterGroupGuid: "test0",
						HostPort:        80,
						HostIP:          "1.2.3.4",
						ExternalPort:    80,
						SniHostname:     &sniHostname1,
					},
				}

				_, err = sqlDB.Client.Create(&tcpRoute1)
				Expect(err).NotTo(HaveOccurred())
			})
			runTests()
		})
	})
})
