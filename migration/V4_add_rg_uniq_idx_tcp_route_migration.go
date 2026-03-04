package migration

import (
	"code.cloudfoundry.org/routing-api/db"
)

type V4AddRgUniqIdxTCPRoute struct{}

var _ Migration = new(V4AddRgUniqIdxTCPRoute)

func NewV4AddRgUniqIdxTCPRouteMigration() *V4AddRgUniqIdxTCPRoute {
	return &V4AddRgUniqIdxTCPRoute{}
}

func (v *V4AddRgUniqIdxTCPRoute) Version() int {
	return 4
}

func (v *V4AddRgUniqIdxTCPRoute) Run(sqlDB *db.SqlDB) error {
	// Drop existing index if it exists - using correct MySQL syntax with table name
	sqlDB.Client.Exec("DROP INDEX IF EXISTS idx_tcp_route ON tcp_routes")

	// Create unique index using raw SQL
	sqlDB.Client.Exec("CREATE UNIQUE INDEX idx_tcp_route ON tcp_routes (router_group_guid, host_port, host_ip, external_port)")

	return nil
}
