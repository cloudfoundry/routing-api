package migration_test

import (
	"fmt"
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

	runTests := func() {
		Context("After migration", func() {
			BeforeEach(func() {
				v8Migration := migration.NewV8HostTLSPortTCPDefaultZero()
				err := v8Migration.Run(sqlDB)
				Expect(err).ToNot(HaveOccurred())

			})

			It("the default value is now 0", func() {
				sniHostname2 := "sniHostname2"
				tcpRoute2 := models.TcpRouteMapping{
					Model:     models.Model{Guid: "guid-3"},
					ExpiresAt: time.Now().Add(1 * time.Hour),
					TcpMappingEntity: models.TcpMappingEntity{
						RouterGroupGuid: "test1-new-host-tls-omitted",
						HostPort:        80,
						HostIP:          "1.2.3.7",
						ExternalPort:    80,
						SniHostname:     &sniHostname2,
					},
				}
				_, err := sqlDB.Client.Create(&tcpRoute2)
				Expect(err).NotTo(HaveOccurred())
				findTcpRouteByRouterGroupGuid(sqlDB, "test1-new-host-tls-omitted", ptrToInt(0))
			})

			It("tcp routes created after the update save their hosttlsport", func() {
				sniHostname1 := "sniHostname1"
				tcpRoute1 := models.TcpRouteMapping{
					Model:     models.Model{Guid: "guid-4"},
					ExpiresAt: time.Now().Add(1 * time.Hour),
					TcpMappingEntity: models.TcpMappingEntity{
						RouterGroupGuid: "test1-new-host-tls-8443",
						HostPort:        80,
						HostTLSPort:     8443,
						HostIP:          "1.2.3.8",
						InstanceId:      "instanceId1",
						ExternalPort:    80,
						SniHostname:     &sniHostname1,
					},
				}
				_, err := sqlDB.Client.Create(&tcpRoute1)
				Expect(err).NotTo(HaveOccurred())
				findTcpRouteByRouterGroupGuid(sqlDB, "test1-new-host-tls-8443", ptrToInt(8443))
			})
		})
	}

	Describe("Version", func() {
		It("returns 8 for the version", func() {
			v8Migration := migration.NewV8HostTLSPortTCPDefaultZero()
			Expect(v8Migration.Version()).To(Equal(8))
		})
	})

	Describe("Run", func() {
		Context("when there are existing tables with NULL host_tls_port values", func() {
			BeforeEach(func() {
				err := sqlDB.Client.AutoMigrate(&v7.RouterGroupDB{}, &v7.TcpRouteMapping{}, &v7.Route{})
				Expect(err).ToNot(HaveOccurred())

				sniHostname1 := "sniHostname1"
				tcpRoute1 := v7.TcpRouteMapping{
					Model:     v7.Model{Guid: "guid-0"},
					ExpiresAt: time.Now().Add(1 * time.Hour),
					TcpMappingEntity: v7.TcpMappingEntity{
						RouterGroupGuid: "test0-preexisting-omitted-host-tls-port",
						HostPort:        80,
						HostIP:          "1.2.3.4",
						ExternalPort:    80,
						SniHostname:     &sniHostname1,
					},
				}
				tcpRoute2 := v7.TcpRouteMapping{
					Model:     v7.Model{Guid: "guid-2"},
					ExpiresAt: time.Now().Add(1 * time.Hour),
					TcpMappingEntity: v7.TcpMappingEntity{
						RouterGroupGuid: "test0-preexisting-host-tls-port-8443",
						HostPort:        80,
						HostTLSPort:     8443,
						HostIP:          "1.2.3.6",
						ExternalPort:    80,
						SniHostname:     &sniHostname1,
					},
				}

				_, err = sqlDB.Client.Create(&tcpRoute1)
				Expect(err).NotTo(HaveOccurred())
				_, err = sqlDB.Client.Create(&tcpRoute2)
				Expect(err).NotTo(HaveOccurred())

				findTcpRouteByRouterGroupGuid(sqlDB, "test0-preexisting-omitted-host-tls-port", nil)
				findTcpRouteByRouterGroupGuid(sqlDB, "test0-preexisting-host-tls-port-8443", ptrToInt(8443))

				v8Migration := migration.NewV8HostTLSPortTCPDefaultZero()
				err = v8Migration.Run(sqlDB)
				Expect(err).ToNot(HaveOccurred())
			})
			runTests()
			It("updates existing records with a NULL value to have a value of 0", func() {
				findTcpRouteByRouterGroupGuid(sqlDB, "test0-preexisting-omitted-host-tls-port", ptrToInt(0))

			})
			It("does not modify existing records with a non-NULL value", func() {
				findTcpRouteByRouterGroupGuid(sqlDB, "test0-preexisting-host-tls-port-8443", ptrToInt(8443))
			})
		})

		Context("when the tables are newly created (by V0 init migration)", func() {
			BeforeEach(func() {
				v0Migration := migration.NewV0InitMigration()
				err := v0Migration.Run(sqlDB)
				Expect(err).ToNot(HaveOccurred())
			})
			runTests()
		})
		Context("when run against a database that was already migrated)", func() {
			BeforeEach(func() {
				err := sqlDB.Client.AutoMigrate(&models.RouterGroupDB{}, &models.TcpRouteMapping{}, &models.Route{})
				Expect(err).ToNot(HaveOccurred())
			})
			runTests()
		})
	})
})

func findTcpRouteByRouterGroupGuid(sqlDB *db.SqlDB, targetRouterGroup string, expectedHostTlsPort *int) {
	rows, err := sqlDB.Client.Rows("tcp_routes")
	Expect(err).ToNot(HaveOccurred())

	rowFound := false

	columns, err := rows.Columns()
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))

	var host_tls_port_column, router_group_guid_column int
	for i, column := range columns {
		switch column {
		case "host_tls_port":
			host_tls_port_column = i
		case "router_group_guid":
			router_group_guid_column = i
		}
	}

	// var args []any
	// columnTypes, err := rows.ColumnTypes()
	// ExpectWithOffset(1, err).NotTo(HaveOccurred())
	// for _, columnType := range columnTypes {
	// 	switch columnType.ScanType() {
	// 	case reflect.PointerTo(reflect.String):
	// 		var str string
	// 		arg = &str
	// 	case *int:
	// 		var i int
	// 		arg = &i
	// 	case reflect.PointerTo(sql.NullDateTime):
	// 		var d sql.NullDateTime
	// 		arg = &d
	// 	case *sql.NullInt64:
	// 		var i sql.NullInt64
	// 		arg = &i
	// 	default:
	// 		panic("Unexpected data type: " + columnType.ScanType())
	// 	}
	// 	args = append(args, val)
	// }

	for rows.Next() {
		for i := range columns {
			valuePtrs[i] = &values[i]
		}
		err = rows.Scan(valuePtrs...)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())

		var router_group string
		host_tls_port := values[host_tls_port_column]

		switch values[router_group_guid_column].(type) {
		case []byte:
			router_group = string(values[router_group_guid_column].([]byte))
		case string:
			router_group = values[router_group_guid_column].(string)
		}
		fmt.Printf("router-group: %#v\n", string(router_group))
		if router_group == targetRouterGroup {
			rowFound = true
			if expectedHostTlsPort == nil {
				ExpectWithOffset(1, host_tls_port).To(BeNil())
			} else {
				ExpectWithOffset(1, host_tls_port).ToNot(BeNil())
				ExpectWithOffset(1, host_tls_port).To(BeNumerically("==", *expectedHostTlsPort))
			}
		}
	}
	By("ensuring we found a record with the right router-group-guid")
	ExpectWithOffset(1, rowFound).To(BeTrue())
}

func ptrToInt(i int) *int {
	return &i
}
