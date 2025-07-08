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

var _ = Describe("V8HostTLSPortTCPDefaultZero", func() {
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

	Describe("Version", func() {
		It("returns 8 for the version", func() {
			v8Migration := migration.NewV8HostTLSPortTCPDefaultZero()
			Expect(v8Migration.Version()).To(Equal(8))
		})
	})

	Describe("Run", func() {
		Context("when a db already exists with values and has not been manually updated", func() {
			BeforeEach(func() {
				err := sqlDB.Client.AutoMigrate(&v7.RouterGroupDB{}, &v7.TcpRouteMapping{}, &v7.Route{})
				Expect(err).ToNot(HaveOccurred())

				sniHostname1 := "sniHostname1"
				tcpRoute1 := v7.TcpRouteMapping{ // This one has no HostTLSPort, before the migration this will default to NULL
					Model:     v7.Model{Guid: "guid-0"},
					ExpiresAt: time.Now().Add(1 * time.Hour),
					TcpMappingEntity: v7.TcpMappingEntity{
						RouterGroupGuid: "test0-preexisting-omitted-host-tls-port",
						HostPort:        80,
						HostIP:          "1.1.1.1",
						ExternalPort:    80,
						SniHostname:     &sniHostname1,
					},
				}

				tcpRoute2 := v7.TcpRouteMapping{ // This one has HostTLSPort set explicitly to 8443
					Model:     v7.Model{Guid: "guid-2"},
					ExpiresAt: time.Now().Add(1 * time.Hour),
					TcpMappingEntity: v7.TcpMappingEntity{
						RouterGroupGuid: "test0-preexisting-host-tls-port-8443",
						HostPort:        80,
						HostTLSPort:     8443,
						HostIP:          "2.2.2.2",
						ExternalPort:    80,
						SniHostname:     &sniHostname1,
					},
				}

				_, err = sqlDB.Client.Create(&tcpRoute1)
				Expect(err).NotTo(HaveOccurred())
				_, err = sqlDB.Client.Create(&tcpRoute2)
				Expect(err).NotTo(HaveOccurred())

				By("validating that there are 2 tcp routes")
				tcpRoutes, err := sqlDB.ReadTcpRouteMappings()
				Expect(err).ToNot(HaveOccurred())
				Expect(len(tcpRoutes)).To(Equal(2))

				By("validating that 1 has host_tls_port set to NULL")
				tcpRoutesWithNULL, err := readFilteredTcpRouteMappingsWhereHostTcpPortIsNull(sqlDB)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(tcpRoutesWithNULL)).To(Equal(1))
				Expect(tcpRoutesWithNULL[0].HostIP).To(Equal("1.1.1.1"))

				By("validating that 1 has host_tls_port set to a non-NULL value")
				tcpRoutesWithoutNULL, err := readFilteredTcpRouteMappingsWhereHostTcpPortIsNotNull(sqlDB)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(tcpRoutesWithoutNULL)).To(Equal(1))
				Expect(tcpRoutesWithoutNULL[0].HostIP).To(Equal("2.2.2.2"))
			})

			It("updates existing records with a NULL value to have a value of 0", func() {
				By("running the migration")
				v8Migration := migration.NewV8HostTLSPortTCPDefaultZero()
				err := v8Migration.Run(sqlDB)
				Expect(err).ToNot(HaveOccurred())

				By("validating that there are still 2 tcp routes")
				tcpRoutes, err := sqlDB.ReadTcpRouteMappings()
				Expect(err).ToNot(HaveOccurred())
				Expect(len(tcpRoutes)).To(Equal(2))

				By("validating that there are now zero tcp routes with host_tls_port set to NULL")
				tcpRoutesWithNULL, err := readFilteredTcpRouteMappingsWhereHostTcpPortIsNull(sqlDB)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(tcpRoutesWithNULL)).To(Equal(0))

				By("validating that there are now two tcp routes with host_tls_port set to a non-NULL value")
				tcpRoutesWithoutNULL, err := readFilteredTcpRouteMappingsWhereHostTcpPortIsNotNull(sqlDB)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(tcpRoutesWithoutNULL)).To(Equal(2))

				By("validating that the host_tls_port for tcpRoute2 did not change")
				tcpRoutes, err = sqlDB.ReadFilteredTcpRouteMappings("host_tls_port", []string{"8443"})
				Expect(err).ToNot(HaveOccurred())
				Expect(len(tcpRoutes)).To(Equal(1))
				Expect(tcpRoutes[0].HostIP).To(Equal("2.2.2.2"))

				By("validating that the host_tls_port for tcpRoute1 is 0 in the db")
				tcpRoutes, err = sqlDB.ReadFilteredTcpRouteMappings("host_tls_port", []string{"0"})
				Expect(err).ToNot(HaveOccurred())
				Expect(len(tcpRoutes)).To(Equal(1))
				Expect(tcpRoutes[0].HostIP).To(Equal("1.1.1.1"))

				By("creating a new route post migration without host_tls_port set")
				tcpRoute3 := v7.TcpRouteMapping{ // This one has no HostTLSPort, before the migration this will default to NULL
					Model:     v7.Model{Guid: "guid-meow"},
					ExpiresAt: time.Now().Add(1 * time.Hour),
					TcpMappingEntity: v7.TcpMappingEntity{
						RouterGroupGuid: "meow-testing-post-migration-when-there-is-no-host-tls-port",
						HostPort:        80,
						HostIP:          "3.3.3.3",
						ExternalPort:    80,
					},
				}
				_, err = sqlDB.Client.Create(&tcpRoute3)
				Expect(err).NotTo(HaveOccurred())

				By("validating that all new tcproutes will default to 0")
				tcpRoutes, err = sqlDB.ReadFilteredTcpRouteMappings("host_tls_port", []string{"0"})
				Expect(err).ToNot(HaveOccurred())
				Expect(len(tcpRoutes)).To(Equal(2))
				Expect([]string{tcpRoutes[0].HostIP, tcpRoutes[1].HostIP}).To(ContainElements("1.1.1.1", "3.3.3.3"))
			})

			Context("when run against a database that was fixed by hand", func() {
				It("doesnt fail during the migration", func() {

					By("manually updating the default")
					if sqlDB.Client.Dialect().GetName() == "postgres" {
						sqlDB.Client.Exec("ALTER TABLE tcp_routes ALTER COLUMN host_tls_port SET DEFAULT 0")
					} else {
						sqlDB.Client.Exec("ALTER TABLE tcp_routes MODIFY COLUMN host_tls_port int DEFAULT 0")
					}
					sqlDB.Client.Exec("UPDATE tcp_routes SET host_tls_port = 0 WHERE host_tls_port IS NULL")

					By("validating that there are still 2 tcp routes")
					tcpRoutes, err := sqlDB.ReadTcpRouteMappings()
					Expect(err).ToNot(HaveOccurred())
					Expect(len(tcpRoutes)).To(Equal(2))

					By("validating that there are now zero tcp routes with host_tls_port set to NULL")
					tcpRoutesWithNULL, err := readFilteredTcpRouteMappingsWhereHostTcpPortIsNull(sqlDB)
					Expect(err).ToNot(HaveOccurred())
					Expect(len(tcpRoutesWithNULL)).To(Equal(0))

					By("validating that there are now two tcp routes with host_tls_port set to a non-NULL value")
					tcpRoutesWithoutNULL, err := readFilteredTcpRouteMappingsWhereHostTcpPortIsNotNull(sqlDB)
					Expect(err).ToNot(HaveOccurred())
					Expect(len(tcpRoutesWithoutNULL)).To(Equal(2))

					By("creating a new route post manual fix without host_tls_port set")
					tcpRoute3 := v7.TcpRouteMapping{ // This one has no HostTLSPort, before the migration this will default to NULL
						Model:     v7.Model{Guid: "guid-meow"},
						ExpiresAt: time.Now().Add(1 * time.Hour),
						TcpMappingEntity: v7.TcpMappingEntity{
							RouterGroupGuid: "meow-testing-post-migration-when-there-is-no-host-tls-port",
							HostPort:        80,
							HostIP:          "3.3.3.3",
							ExternalPort:    80,
						},
					}
					_, err = sqlDB.Client.Create(&tcpRoute3)
					Expect(err).NotTo(HaveOccurred())

					By("validating that new tcproutes will default to 0")
					tcpRoutes, err = sqlDB.ReadFilteredTcpRouteMappings("host_tls_port", []string{"0"})
					Expect(err).ToNot(HaveOccurred())
					Expect(len(tcpRoutes)).To(Equal(2))
					Expect([]string{tcpRoutes[0].HostIP, tcpRoutes[1].HostIP}).To(ContainElements("1.1.1.1", "3.3.3.3"))

					By("running the migration")
					v8Migration := migration.NewV8HostTLSPortTCPDefaultZero()
					err = v8Migration.Run(sqlDB)
					Expect(err).ToNot(HaveOccurred())

					By("validating that there are still 3 tcp routes")
					tcpRoutes, err = sqlDB.ReadTcpRouteMappings()
					Expect(err).ToNot(HaveOccurred())
					Expect(len(tcpRoutes)).To(Equal(3))

					By("validating that there are now zero tcp routes with host_tls_port set to NULL")
					tcpRoutesWithNULL, err = readFilteredTcpRouteMappingsWhereHostTcpPortIsNull(sqlDB)
					Expect(err).ToNot(HaveOccurred())
					Expect(len(tcpRoutesWithNULL)).To(Equal(0))

					By("validating that there are now two tcp routes with host_tls_port set to a non-NULL value")
					tcpRoutesWithoutNULL, err = readFilteredTcpRouteMappingsWhereHostTcpPortIsNotNull(sqlDB)
					Expect(err).ToNot(HaveOccurred())
					Expect(len(tcpRoutesWithoutNULL)).To(Equal(3))

					By("validating that the host_tls_port for tcpRoute2 did not change")
					tcpRoutes, err = sqlDB.ReadFilteredTcpRouteMappings("host_tls_port", []string{"8443"})
					Expect(err).ToNot(HaveOccurred())
					Expect(len(tcpRoutes)).To(Equal(1))
					Expect(tcpRoutes[0].HostIP).To(Equal("2.2.2.2"))

					By("creating a new route post migration without host_tls_port set")
					tcpRoute4 := v7.TcpRouteMapping{ // This one has no HostTLSPort, before the migration this will default to NULL
						Model:     v7.Model{Guid: "guid-meow-4"},
						ExpiresAt: time.Now().Add(1 * time.Hour),
						TcpMappingEntity: v7.TcpMappingEntity{
							RouterGroupGuid: "meow-testing-post-migration-when-there-is-no-host-tls-port-4",
							HostPort:        44,
							HostIP:          "4.4.4.4",
							ExternalPort:    44,
						},
					}
					_, err = sqlDB.Client.Create(&tcpRoute4)
					Expect(err).NotTo(HaveOccurred())

					By("validating that all tcproutes will still default to 0")
					tcpRoutes, err = sqlDB.ReadFilteredTcpRouteMappings("host_tls_port", []string{"0"})
					Expect(err).ToNot(HaveOccurred())
					Expect(len(tcpRoutes)).To(Equal(3))
					Expect([]string{tcpRoutes[0].HostIP, tcpRoutes[1].HostIP, tcpRoutes[2].HostIP}).To(ContainElements("1.1.1.1", "3.3.3.3", "4.4.4.4"))
					// 1.1.1.1 was made before fixing by hand
					// 3.3.3.3 was made after the manual fix
					// 4.4.4.4 was made after the migration
				})
			})
		})

		Context("when the tables are newly created (by V0 init migration)", func() {
			BeforeEach(func() {
				v0Migration := migration.NewV0InitMigration()
				err := v0Migration.Run(sqlDB)
				Expect(err).ToNot(HaveOccurred())

				By("running the migration")
				v8Migration := migration.NewV8HostTLSPortTCPDefaultZero()
				err = v8Migration.Run(sqlDB)
				Expect(err).ToNot(HaveOccurred())
			})

			It("always has default 0 for host_tls_port from the beginning", func() {
				By("creating a new route post migration without host_tls_port set")
				tcpRoute := v7.TcpRouteMapping{ // This one has no HostTLSPort, before the migration this will default to NULL
					Model:     v7.Model{Guid: "guid-meow"},
					ExpiresAt: time.Now().Add(1 * time.Hour),
					TcpMappingEntity: v7.TcpMappingEntity{
						RouterGroupGuid: "meow-testing-post-migration-when-there-is-no-host-tls-port",
						HostPort:        80,
						HostIP:          "1.1.1.1",
						ExternalPort:    80,
					},
				}
				_, err := sqlDB.Client.Create(&tcpRoute)
				Expect(err).NotTo(HaveOccurred())

				By("validating that all new tcproutes will default to 0")
				tcpRoutes, err := sqlDB.ReadFilteredTcpRouteMappings("host_tls_port", []string{"0"})
				Expect(err).ToNot(HaveOccurred())
				Expect(len(tcpRoutes)).To(Equal(1))
				Expect(tcpRoutes[0].HostIP).To(Equal("1.1.1.1"))

				By("validating that there are zero tcp routes with host_tls_port set to NULL")
				tcpRoutesWithNULL, err := readFilteredTcpRouteMappingsWhereHostTcpPortIsNull(sqlDB)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(tcpRoutesWithNULL)).To(Equal(0))
			})
		})

		Context("when run against a database that was already migrated", func() {
			BeforeEach(func() {
				err := sqlDB.Client.AutoMigrate(&models.RouterGroupDB{}, &models.TcpRouteMapping{}, &models.Route{})
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})

func readFilteredTcpRouteMappingsWhereHostTcpPortIsNull(s *db.SqlDB) ([]models.TcpRouteMapping, error) {
	var tcpRoutes []models.TcpRouteMapping
	now := time.Now()
	err := s.Client.Where("host_tls_port IS NULL").Where("expires_at > ?", now).Find(&tcpRoutes)
	if err != nil {
		return nil, err
	}
	return tcpRoutes, nil
}

func readFilteredTcpRouteMappingsWhereHostTcpPortIsNotNull(s *db.SqlDB) ([]models.TcpRouteMapping, error) {
	var tcpRoutes []models.TcpRouteMapping
	now := time.Now()
	err := s.Client.Where("host_tls_port IS NOT NULL").Where("expires_at > ?", now).Find(&tcpRoutes)
	if err != nil {
		return nil, err
	}
	return tcpRoutes, nil
}
