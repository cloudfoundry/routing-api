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

var _ = Describe("V11EnableBackendMTLS", func() {
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
				v11Migration := migration.NewV11EnableBackendMTLS()
				err := v11Migration.Run(sqlDB)
				Expect(err).ToNot(HaveOccurred())

				routes, err := sqlDB.ReadTcpRouteMappings()
				Expect(err).NotTo(HaveOccurred())
				Expect(routes).To(HaveLen(1))
				// The existing route should default to EnableBackendMTLS=false after migration
				Expect(routes[0].EnableBackendMTLS).To(Equal(false))
			})
		})
		Context("After migration", func() {
			var tcpRoute1 models.TcpRouteMapping

			BeforeEach(func() {
				v11Migration := migration.NewV11EnableBackendMTLS()
				err := v11Migration.Run(sqlDB)
				Expect(err).ToNot(HaveOccurred())

				sniHostname1 := "sniHostname1"
				tcpRoute1 = models.TcpRouteMapping{
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
						TerminateFrontendTLS: false,
						ALPNs:                "",
						EnableBackendMTLS:    true,
					},
				}
			})

			It("expect no error to occur when creating a route with enable_backend_mtls", func() {
				_, err := sqlDB.Client.Create(&tcpRoute1)
				Expect(err).NotTo(HaveOccurred())

				var createdRoute models.TcpRouteMapping
				err = sqlDB.Client.Where("guid = ?", "guid-1").First(&createdRoute)
				Expect(err).NotTo(HaveOccurred())
				Expect(createdRoute.EnableBackendMTLS).To(Equal(true))
			})

			It("allows creating a route without enable_backend_mtls (false)", func() {
				tcpRoute1.EnableBackendMTLS = false
				_, err := sqlDB.Client.Create(&tcpRoute1)
				Expect(err).NotTo(HaveOccurred())

				var createdRoute models.TcpRouteMapping
				err = sqlDB.Client.Where("guid = ?", "guid-1").First(&createdRoute)
				Expect(err).NotTo(HaveOccurred())
				Expect(createdRoute.EnableBackendMTLS).To(Equal(false))
			})

			It("allows two otherwise-identical routes that differ only by enable_backend_mtls", func() {
				_, err := sqlDB.Client.Create(&tcpRoute1)
				Expect(err).NotTo(HaveOccurred())

				sibling := tcpRoute1
				sibling.Model = models.Model{Guid: "guid-2"}
				sibling.EnableBackendMTLS = false

				_, err = sqlDB.Client.Create(&sibling)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	}

	Describe("Version", func() {
		It("returns 11 for the version", func() {
			v11Migration := migration.NewV11EnableBackendMTLS()
			Expect(v11Migration.Version()).To(Equal(11))
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

				v11Migration := migration.NewV11EnableBackendMTLS()
				err = v11Migration.Run(sqlDB)
				Expect(err).ToNot(HaveOccurred())
			})
			runTests()
		})

		Context("when the tables are newly created (by V0 init migration)", func() {
			BeforeEach(func() {
				v0Migration := migration.NewV0InitMigration()
				err := v0Migration.Run(sqlDB)
				Expect(err).ToNot(HaveOccurred())

				// Run all migrations up to V10
				v9Migration := migration.NewV9TerminateFrontendTLS()
				err = v9Migration.Run(sqlDB)
				Expect(err).ToNot(HaveOccurred())

				v10Migration := migration.NewV10SniRewriteHostname()
				err = v10Migration.Run(sqlDB)
				Expect(err).ToNot(HaveOccurred())

				sniHostname1 := "sniHostname1"
				tcpRoute1 := models.TcpRouteMapping{
					Model:     models.Model{Guid: "guid-0"},
					ExpiresAt: time.Now().Add(1 * time.Hour),
					TcpMappingEntity: models.TcpMappingEntity{
						RouterGroupGuid:      "test0",
						HostPort:             80,
						HostTLSPort:          100,
						HostIP:               "1.2.3.4",
						SniHostname:          &sniHostname1,
						InstanceId:           "",
						ExternalPort:         80,
						ModificationTag:      models.ModificationTag{},
						TTL:                  nil,
						IsolationSegment:     "",
						TerminateFrontendTLS: false,
						ALPNs:                "",
					},
				}

				_, err = sqlDB.Client.Create(&tcpRoute1)
				Expect(err).NotTo(HaveOccurred())
			})
			runTests()
		})
	})
})
