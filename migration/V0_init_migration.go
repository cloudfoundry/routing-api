package migration

import (
	"code.cloudfoundry.org/routing-api/db"
	"code.cloudfoundry.org/routing-api/models"
	"github.com/jinzhu/gorm"
)

type V0InitMigration struct{}

var _ Migration = new(V0InitMigration)

func NewV0InitMigration() *V0InitMigration {
	return &V0InitMigration{}
}

func (v *V0InitMigration) Version() int {
	return 0
}

func (v *V0InitMigration) Run(sqlDB *db.SqlDB) error {
	gormDB := sqlDB.Client.(*gorm.DB)
	return gormDB.AutoMigrate(&models.RouterGroupDB{}, &models.TcpRouteMapping{}, &models.Route{}).Error
}
