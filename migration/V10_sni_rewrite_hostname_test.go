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

var _ = Describe("V10SniRewriteHostname", func() {
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
				v10Migration := migration.NewV10SniRewriteHostname()
				err := v10Migration.Run(sqlDB)
				Expect(err).ToNot(HaveOccurred())

				routes, err := sqlDB.ReadTcpRouteMappings()
				Expect(err).NotTo(HaveOccurred())
				Expect(routes).To(HaveLen(1))
				// The existing route should have nil sni_rewrite_hostname after migration
				Expect(routes[0].SniRewriteHostname).To(BeNil())
			})
		})
		Context("After migration", func() {
			var tcpRoute1 models.TcpRouteMapping

			BeforeEach(func() {
				v10Migration := migration.NewV10SniRewriteHostname()
				err := v10Migration.Run(sqlDB)
				Expect(err).ToNot(HaveOccurred())

				sniHostname1 := "sniHostname1"
				sniRewriteHostname1 := "sniRewriteHostname1"
				tcpRoute1 = models.TcpRouteMapping{
					Model:     models.Model{Guid: "guid-1"},
					ExpiresAt: time.Now().Add(1 * time.Hour),
					TcpMappingEntity: models.TcpMappingEntity{
						RouterGroupGuid:    "test1",
						HostPort:           80,
						HostTLSPort:        443,
						HostIP:             "1.2.3.4",
						InstanceId:         "instanceId1",
						ExternalPort:       80,
						SniHostname:        &sniHostname1,
						SniRewriteHostname: &sniRewriteHostname1,

						ModificationTag:      models.ModificationTag{},
						TTL:                  nil,
						IsolationSegment:     "",
						TerminateFrontendTLS: false,
						ALPNs:                "",
					},
				}
			})

			It("expect no error to occur when creating route with sni_rewrite_hostname", func() {
				_, err := sqlDB.Client.Create(&tcpRoute1)
				Expect(err).NotTo(HaveOccurred())

				// Verify the route was created with sni_rewrite_hostname
				routes, err := sqlDB.ReadTcpRouteMappings()
				Expect(err).NotTo(HaveOccurred())
				Expect(routes).To(HaveLen(1))
				Expect(routes[0].SniRewriteHostname).ToNot(BeNil())
				Expect(*routes[0].SniRewriteHostname).To(Equal("sniRewriteHostname1"))
			})

			It("allows creating route without sni_rewrite_hostname (nil)", func() {
				tcpRoute1.SniRewriteHostname = nil
				_, err := sqlDB.Client.Create(&tcpRoute1)
				Expect(err).NotTo(HaveOccurred())

				routes, err := sqlDB.ReadTcpRouteMappings()
				Expect(err).NotTo(HaveOccurred())
				Expect(routes).To(HaveLen(1))
				Expect(routes[0].SniRewriteHostname).To(BeNil())
			})
		})
	}

	Describe("Version", func() {
		It("returns 10 for the version", func() {
			v10Migration := migration.NewV10SniRewriteHostname()
			Expect(v10Migration.Version()).To(Equal(10))
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

				v10Migration := migration.NewV10SniRewriteHostname()
				err = v10Migration.Run(sqlDB)
				Expect(err).ToNot(HaveOccurred())
			})
			runTests()
		})

		Context("when the tables are newly created (by V0 init migration)", func() {
			BeforeEach(func() {
				v0Migration := migration.NewV0InitMigration()
				err := v0Migration.Run(sqlDB)
				Expect(err).ToNot(HaveOccurred())

				// Run all migrations up to V9
				v9Migration := migration.NewV9TerminateFrontendTLS()
				err = v9Migration.Run(sqlDB)
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
