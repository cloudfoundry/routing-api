package migration

import (
	"code.cloudfoundry.org/routing-api/db"
)

type V8HostTLSPortTCPDefaultZero struct{}

func NewV8HostTLSPortTCPDefaultZero() *V8HostTLSPortTCPDefaultZero {
	return &V8HostTLSPortTCPDefaultZero{}
}

func (v *V8HostTLSPortTCPDefaultZero) Version() int {
	return 8
}

func (v *V8HostTLSPortTCPDefaultZero) Run(sqlDB *db.SqlDB) error {
	// Update existing rows where host_tls_port is NULL to 0
	err := sqlDB.Client.ExecWithError("UPDATE tcp_routes SET host_tls_port = 0 WHERE host_tls_port IS NULL")
	if err != nil {
		return err
	}

	// Try to remove the old index if it exists - using correct MySQL syntax with table name
	_ = sqlDB.Client.ExecWithError("DROP INDEX IF EXISTS idx_tcp_route ON tcp_routes")

	// Set the DEFAULT 0 on the host_tls_port column
	err = sqlDB.Client.ExecWithError("ALTER TABLE tcp_routes MODIFY COLUMN host_tls_port int DEFAULT 0")
	if err != nil {
		return err
	}

	return nil
}
