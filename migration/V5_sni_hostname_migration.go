package migration

import (
	"code.cloudfoundry.org/routing-api/db"
	"code.cloudfoundry.org/routing-api/models"
)

type V5SniHostnameMigration struct{}

var _ Migration = new(V5SniHostnameMigration)

func NewV5SniHostnameMigration() *V5SniHostnameMigration {
	return &V5SniHostnameMigration{}
}

func (v *V5SniHostnameMigration) Version() int {
	return 5
}

func (v *V5SniHostnameMigration) Run(sqlDB *db.SqlDB) error {
	err := sqlDB.Client.AutoMigrate(&models.TcpRouteMapping{})
	if err != nil {
		return err
	}

	// Drop old index if it exists (ignore errors since it might not exist)
	dropIndex(sqlDB, "idx_tcp_route", "tcp_routes")

	// Create unique index - MySQL requires length prefixes for text columns
	var indexSQL string
	if sqlDB.Client.Dialect().Name() == "mysql" {
		indexSQL = "CREATE UNIQUE INDEX idx_tcp_route ON tcp_routes (router_group_guid(191), host_port, host_ip(191), external_port, sni_hostname(191))"
	} else {
		indexSQL = "CREATE UNIQUE INDEX idx_tcp_route ON tcp_routes (router_group_guid, host_port, host_ip, external_port, sni_hostname)"
	}
	return sqlDB.Client.ExecWithError(indexSQL)
}
