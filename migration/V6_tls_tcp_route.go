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

	dropIndex(sqlDB, "idx_tcp_route", "tcp_routes")

	// Create unique index - MySQL requires length prefixes for text columns
	var indexSQL string
	if sqlDB.Client.Dialect().Name() == "mysql" {
		indexSQL = "CREATE UNIQUE INDEX idx_tcp_route ON tcp_routes (router_group_guid(191), host_port, host_ip(191), external_port, sni_hostname(191), host_tls_port)"
	} else {
		indexSQL = "CREATE UNIQUE INDEX idx_tcp_route ON tcp_routes (router_group_guid, host_port, host_ip, external_port, sni_hostname, host_tls_port)"
	}
	return sqlDB.Client.ExecWithError(indexSQL)
}
