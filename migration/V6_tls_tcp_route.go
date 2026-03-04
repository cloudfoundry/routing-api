package migration

import (
	"code.cloudfoundry.org/routing-api/db"
	"code.cloudfoundry.org/routing-api/models"
)

type V6TCPTLSRoutes struct{}

var _ Migration = new(V6TCPTLSRoutes)

func NewV6TCPTLSRoutes() *V6TCPTLSRoutes {
	return &V6TCPTLSRoutes{}
}

func (v *V6TCPTLSRoutes) Version() int {
	return 6
}

func (v *V6TCPTLSRoutes) Run(sqlDB *db.SqlDB) error {
	err := sqlDB.Client.AutoMigrate(&models.TcpRouteMapping{})
	if err != nil {
		return err
	}

	// Drop existing index if it exists - using correct MySQL syntax with table name
	_ = sqlDB.Client.ExecWithError("DROP INDEX idx_tcp_route ON tcp_routes")

	// Create unique index using raw SQL with column length specifications
	err = sqlDB.Client.ExecWithError("CREATE UNIQUE INDEX idx_tcp_route ON tcp_routes (router_group_guid(191), host_port, host_ip(191), external_port, sni_hostname(191), host_tls_port)")
	if err != nil {
		return err
	}

	return nil
}
