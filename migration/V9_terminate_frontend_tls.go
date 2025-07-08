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
	return 6
}

func (v *V9TerminateFrontendTLS) Run(sqlDB *db.SqlDB) error {
	_, err := sqlDB.Client.Model(&models.TcpRouteMapping{}).RemoveIndex("idx_tcp_route")
	if err != nil {
		return err
	}
	err = sqlDB.Client.AutoMigrate(&models.TcpRouteMapping{})
	if err != nil {
		return err
	}
	_, err = sqlDB.Client.Model(&models.TcpRouteMapping{}).AddUniqueIndex("idx_tcp_route", "router_group_guid", "host_port", "host_ip", "external_port", "sni_hostname", "host_tls_port", "terminate_frontend_tls")
	if err != nil {
		return err
	}
	return err
}
