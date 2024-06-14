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
	err := sqlDB.Client.RemoveIndex("idx_tcp_route", &models.TcpRouteMapping{})
	if err != nil {
		return err
	}
	_, err = sqlDB.Client.Model(&models.TcpRouteMapping{}).AddUniqueIndex("idx_tcp_route", "router_group_guid", "host_port", "host_ip", "external_port", "sni_hostname")
	if err != nil {
		return err
	}
	return err
}
