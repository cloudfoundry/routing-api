package migration

import (
	"code.cloudfoundry.org/routing-api/db"
	"code.cloudfoundry.org/routing-api/models"
)

type V8HostTLSPortTCPDefaultZero struct{}

func NewV8HostTLSPortTCPDefaultZero() *V8HostTLSPortTCPDefaultZero {
	return &V8HostTLSPortTCPDefaultZero{}
}

func (v *V8HostTLSPortTCPDefaultZero) Version() int {
	return 8
}

func (v *V8HostTLSPortTCPDefaultZero) Run(sqlDB *db.SqlDB) error {
	_, err := sqlDB.Client.Model(&models.TcpRouteMapping{}).RemoveIndex("idx_tcp_route")
	if err != nil {
		return err
	}

	if sqlDB.Client.Dialect().GetName() == "postgres" {
		sqlDB.Client.Exec("ALTER TABLE tcp_routes ALTER COLUMN host_tls_port SET DEFAULT 0")
	} else {
		sqlDB.Client.Exec("ALTER TABLE tcp_routes MODIFY COLUMN host_tls_port int DEFAULT 0")
	}

	sqlDB.Client.Exec("UPDATE tcp_routes SET host_tls_port = 0 WHERE host_tls_port IS NULL")

	return nil
}
