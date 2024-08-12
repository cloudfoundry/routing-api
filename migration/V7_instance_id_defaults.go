package migration

import (
	"code.cloudfoundry.org/routing-api/db"
	"code.cloudfoundry.org/routing-api/models"
)

type V7TCPTLSRoutes struct{}

func NewV7TCPTLSRoutes() *V7TCPTLSRoutes {
	return &V7TCPTLSRoutes{}
}

func (v *V7TCPTLSRoutes) Version() int {
	return 7
}

func (v *V7TCPTLSRoutes) Run(sqlDB *db.SqlDB) error {
	_, err := sqlDB.Client.Model(&models.TcpRouteMapping{}).RemoveIndex("idx_tcp_route")
	if err != nil {
		return err
	}

	if sqlDB.Client.Dialect().GetName() == "postgres" {
		sqlDB.Client.Exec("ALTER TABLE tcp_routes ALTER COLUMN instance_id DROP NOT NULL")
	} else {
		sqlDB.Client.Exec("ALTER TABLE tcp_routes MODIFY COLUMN instance_id varchar(255) DEFAULT NULL")
	}

	_, err = sqlDB.Client.Model(&models.TcpRouteMapping{}).AddUniqueIndex("idx_tcp_route", "router_group_guid", "host_port", "host_ip", "external_port", "sni_hostname", "host_tls_port")
	if err != nil {
		return err
	}
	return err
}
