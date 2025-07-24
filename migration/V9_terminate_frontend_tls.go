package migration

import (
	"code.cloudfoundry.org/routing-api/db"
	"code.cloudfoundry.org/routing-api/models"
)

type V9TerminateFrontendTLS struct{}

var _ Migration = new(V9TerminateFrontendTLS)

func NewV9TerminateFrontendTLS() *V9TerminateFrontendTLS {
	return &V9TerminateFrontendTLS{}
}

func (v *V9TerminateFrontendTLS) Version() int {
	return 9
}

func (v *V9TerminateFrontendTLS) Run(sqlDB *db.SqlDB) error {
	return sqlDB.Client.AutoMigrate(&models.TcpRouteMapping{})
}
