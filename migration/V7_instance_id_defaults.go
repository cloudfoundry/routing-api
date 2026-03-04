package migration

import (
	"code.cloudfoundry.org/routing-api/db"
)

type V7TCPTLSRoutes struct{}

func NewV7TCPTLSRoutes() *V7TCPTLSRoutes {
	return &V7TCPTLSRoutes{}
}

func (v *V7TCPTLSRoutes) Version() int {
	return 7
}

func (v *V7TCPTLSRoutes) Run(sqlDB *db.SqlDB) error {
	// Update the instance_id column to allow NULL values
	err := sqlDB.Client.ExecWithError("ALTER TABLE tcp_routes MODIFY COLUMN instance_id varchar(255) DEFAULT NULL")
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
