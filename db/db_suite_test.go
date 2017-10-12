package db_test

import (
	"testing"

	"code.cloudfoundry.org/routing-api/cmd/routing-api/testrunner"
	"code.cloudfoundry.org/routing-api/config"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var sqlCfg *config.SqlDB
var sqlDBName string
var postgresDBName string
var mysqlAllocator testrunner.DbAllocator
var postgresAllocator testrunner.DbAllocator

func TestDB(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DB Suite")
}

var _ = BeforeSuite(func() {
	var err error

	postgresAllocator = testrunner.NewPostgresAllocator()
	postgresDBName, err = postgresAllocator.Create()
	Expect(err).ToNot(HaveOccurred())

	mysqlAllocator = testrunner.NewMySQLAllocator()
	sqlDBName, err = mysqlAllocator.Create()
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {

	err := mysqlAllocator.Delete()
	Expect(err).ToNot(HaveOccurred())

	err = postgresAllocator.Delete()
	Expect(err).ToNot(HaveOccurred())
})

var _ = BeforeEach(func() {
	err := mysqlAllocator.Reset()
	Expect(err).ToNot(HaveOccurred())
	err = postgresAllocator.Reset()
	Expect(err).ToNot(HaveOccurred())
})
