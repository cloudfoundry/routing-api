package db_test

import (
	"testing"

	"code.cloudfoundry.org/routing-api/cmd/routing-api/testrunner"
	"code.cloudfoundry.org/routing-api/config"
	_ "github.com/lib/pq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	databaseCfg       *config.SqlDB
	databaseAllocator testrunner.DbAllocator
)

func TestDB(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DB Suite")
}

var _ = BeforeSuite(func() {
	var err error

	databaseAllocator = testrunner.NewDbAllocator()
	databaseCfg, err = databaseAllocator.Create()
	Expect(err).ToNot(HaveOccurred(), "error occurred starting database client, is the database running?")
})

var _ = AfterSuite(func() {
	err := databaseAllocator.Delete()
	Expect(err).ToNot(HaveOccurred())
})

var _ = BeforeEach(func() {
	err := databaseAllocator.Reset()
	Expect(err).ToNot(HaveOccurred())
})
