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
		sqlDB.Client.Exec("ALTER TABLE tcp_routes ALTER COLUMN host_tls_port int DEFAULT 0")
	} else {
		sqlDB.Client.Exec("ALTER TABLE tcp_routes MODIFY COLUMN host_tls_port int DEFAULT 0")
	}

	_, err = sqlDB.Client.Model(&models.TcpRouteMapping{}).AddUniqueIndex("idx_tcp_route", "router_group_guid", "host_port", "host_ip", "external_port", "sni_hostname", "host_tls_port")
	if err != nil {
		return err
	}

	routesToUpdate := []*models.TcpRouteMapping{}
	err = sqlDB.Client.Where("host_tls_port IS NULL").Find(routesToUpdate)
	if err != nil {
		return err
	}
	for _, route := range routesToUpdate {
		route.HostTLSPort = 0
		_, err = sqlDB.Client.Save(route)
		if err != nil {
			return err
		}
	}
	return nil
}
