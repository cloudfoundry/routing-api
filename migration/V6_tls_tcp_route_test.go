package migration_test

import (
	"time"

	"code.cloudfoundry.org/routing-api/cmd/routing-api/testrunner"
	"code.cloudfoundry.org/routing-api/db"
	"code.cloudfoundry.org/routing-api/migration"
	v5 "code.cloudfoundry.org/routing-api/migration/v5"
	"code.cloudfoundry.org/routing-api/models"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("V6TCPTLSRoutes", func() {
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
		Context("After migration", func() {
			BeforeEach(func() {
				v6Migration := migration.NewV6TCPTLSRoutes()
				err := v6Migration.Run(sqlDB)
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
					},
				}
				_, err = sqlDB.Client.Create(&tcpRoute1)
				Expect(err).NotTo(HaveOccurred())
			})

			It("allows adding the same TCP routes with different host TLS ports", func() {
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
				Expect(routes).To(HaveLen(2))
			})

			It("denies adding the same TCP routes with same host TLS ports", func() {
				sniHostname1 := "sniHostname1"
				tcpRoute2 := models.TcpRouteMapping{
					Model:     models.Model{Guid: "guid-2"},
					ExpiresAt: time.Now().Add(1 * time.Hour),
					TcpMappingEntity: models.TcpMappingEntity{
						RouterGroupGuid: "test1",
						HostPort:        80,
						HostTLSPort:     443,
						InstanceId:      "instanceId2",
						HostIP:          "1.2.3.4",
						ExternalPort:    80,
						SniHostname:     &sniHostname1,
					},
				}
				_, err := sqlDB.Client.Create(&tcpRoute2)
				Expect(err).To(HaveOccurred())

				routes, err := sqlDB.ReadTcpRouteMappings()
				Expect(err).NotTo(HaveOccurred())
				Expect(routes).To(HaveLen(1))
			})

			It("denies adding the same TCP routes with different instance_ids", func() {
				sniHostname1 := "sniHostname1"
				tcpRoute2 := models.TcpRouteMapping{
					Model:     models.Model{Guid: "guid-2"},
					ExpiresAt: time.Now().Add(1 * time.Hour),
					TcpMappingEntity: models.TcpMappingEntity{
						RouterGroupGuid: "test1",
						HostPort:        80,
						HostTLSPort:     443,
						HostIP:          "1.2.3.4",
						InstanceId:      "instanceId2",
						ExternalPort:    80,
						SniHostname:     &sniHostname1,
					},
				}
				_, err := sqlDB.Client.Create(&tcpRoute2)
				Expect(err).To(HaveOccurred())

				routes, err := sqlDB.ReadTcpRouteMappings()
				Expect(err).NotTo(HaveOccurred())
				Expect(routes).To(HaveLen(1))
			})
		})
	}

	Describe("Version", func() {
		It("returns 6 for the version", func() {
			v6Migration := migration.NewV6TCPTLSRoutes()
			Expect(v6Migration.Version()).To(Equal(6))
		})
	})

	Describe("Run", func() {
		Context("when there are existing tables with the old tcp_route model", func() {
			BeforeEach(func() {
				err := sqlDB.Client.AutoMigrate(&v5.RouterGroupDB{}, &v5.TcpRouteMapping{}, &v5.Route{})
				Expect(err).ToNot(HaveOccurred())
				v5Migration := migration.NewV5SniHostnameMigration()
				err = v5Migration.Run(sqlDB)
				Expect(err).ToNot(HaveOccurred())
			})
			runTests()
		})

		Context("when the tables are newly created (by V0 init migration)", func() {
			BeforeEach(func() {
				v0Migration := migration.NewV0InitMigration()
				err := v0Migration.Run(sqlDB)
				Expect(err).ToNot(HaveOccurred())
			})
			runTests()
		})
	})
})
