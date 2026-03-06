package migration_test

import (
	"testing"

	"code.cloudfoundry.org/routing-api/cmd/routing-api/testrunner"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	databaseAllocator testrunner.DbAllocator
)

func TestMigration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Migration Suite")
}

var _ = BeforeSuite(func() {
	var err error
	databaseAllocator = testrunner.NewDbAllocator()
	_, err = databaseAllocator.Create()
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
