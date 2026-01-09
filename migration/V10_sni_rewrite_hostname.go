package migration

import (
	"code.cloudfoundry.org/routing-api/db"
	"code.cloudfoundry.org/routing-api/models"
)

type V10SniRewriteHostname struct{}

var _ Migration = new(V10SniRewriteHostname)

func NewV10SniRewriteHostname() *V10SniRewriteHostname {
	return &V10SniRewriteHostname{}
}

func (v *V10SniRewriteHostname) Version() int {
	return 10
}

func (v *V10SniRewriteHostname) Run(sqlDB *db.SqlDB) error {
	return sqlDB.Client.AutoMigrate(&models.TcpRouteMapping{})
}
