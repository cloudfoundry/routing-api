package migration

import (
	"code.cloudfoundry.org/routing-api/db"
	"code.cloudfoundry.org/routing-api/models"
)

type V11EnableBackendMTLS struct{}

var _ Migration = new(V11EnableBackendMTLS)

func NewV11EnableBackendMTLS() *V11EnableBackendMTLS {
	return &V11EnableBackendMTLS{}
}

func (v *V11EnableBackendMTLS) Version() int {
	return 11
}

func (v *V11EnableBackendMTLS) Run(sqlDB *db.SqlDB) error {
	// Drop index BEFORE AutoMigrate to avoid MySQL error 1170
	// when Gorm v2 tries to change VARCHAR columns to LONGTEXT
	dropIndex(sqlDB, "idx_tcp_route", "tcp_routes")

	// Run AutoMigrate to add the EnableBackendMTLS column
	err := sqlDB.Client.AutoMigrate(&models.TcpRouteMapping{})
	if err != nil {
		return err
	}

	// Recreate unique index with proper MySQL prefix lengths for LONGTEXT columns
	var indexSQL string
	if sqlDB.Client.Dialect().Name() == "mysql" {
		// MySQL requires prefix lengths for TEXT/LONGTEXT columns in indexes
		indexSQL = "CREATE UNIQUE INDEX idx_tcp_route ON tcp_routes (router_group_guid(191), host_port, host_ip(191), external_port, sni_hostname(191), host_tls_port, terminate_frontend_tls, enable_backend_m_tls)"
	} else {
		// PostgreSQL doesn't require prefix lengths
		indexSQL = "CREATE UNIQUE INDEX idx_tcp_route ON tcp_routes (router_group_guid, host_port, host_ip, external_port, sni_hostname, host_tls_port, terminate_frontend_tls, enable_backend_m_tls)"
	}
	return sqlDB.Client.ExecWithError(indexSQL)
}
