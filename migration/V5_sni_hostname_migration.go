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
	_ = sqlDB.Client.ExecWithError("DROP INDEX idx_tcp_route ON tcp_routes")

	// Create unique index for SNI hostname constraint with column length specifications
	// This allows routes with same fingerprint but different SNI hostnames
	err = sqlDB.Client.ExecWithError("CREATE UNIQUE INDEX idx_tcp_route ON tcp_routes (router_group_guid(191), host_port, host_ip(191), external_port, sni_hostname(191))")
	if err != nil {
		return err
	}

	return nil
}
