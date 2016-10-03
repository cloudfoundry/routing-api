package migration_test

import (
	"code.cloudfoundry.org/routing-api/cmd/routing-api/testrunner"
	"code.cloudfoundry.org/routing-api/config"
	"code.cloudfoundry.org/routing-api/db"
	"code.cloudfoundry.org/routing-api/migration"
	"code.cloudfoundry.org/routing-api/models"

	"github.com/jinzhu/gorm"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("V0InitMigration", func() {
	var (
		mysqlAllocator testrunner.DbAllocator
		gormDb         *gorm.DB
		sqlDB          *db.SqlDB
		err            error
	)
	BeforeEach(func() {
		mysqlAllocator = testrunner.NewMySQLAllocator()
		mysqlSchema, err := mysqlAllocator.Create()
		Expect(err).NotTo(HaveOccurred())

		sqlCfg := &config.SqlDB{
			Username: "root",
			Password: "password",
			Schema:   mysqlSchema,
			Host:     "localhost",
			Port:     3306,
			Type:     "mysql",
		}

		sqlDB, err = db.NewSqlDB(sqlCfg)
		Expect(err).ToNot(HaveOccurred())
		gormDb = sqlDB.Client.(*gorm.DB)
	})

	AfterEach(func() {
		mysqlAllocator.Delete()
	})

	Context("when valid sql config is passed", func() {
		var v0Migration *migration.V0InitMigration
		BeforeEach(func() {
			v0Migration = migration.NewV0InitMigration()
		})

		It("should successfully create correct schema and does not close db connection", func() {
			err = v0Migration.Run(sqlDB)
			Expect(err).ToNot(HaveOccurred())

			Expect(gormDb.HasTable(&models.RouterGroupDB{})).To(BeTrue())
			Expect(gormDb.HasTable(&models.TcpRouteMapping{})).To(BeTrue())
			Expect(gormDb.HasTable(&models.Route{})).To(BeTrue())
		})
	})
})
