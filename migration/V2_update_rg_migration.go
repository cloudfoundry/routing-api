package migration

import (
	"fmt"

	"code.cloudfoundry.org/routing-api/db"
	"code.cloudfoundry.org/routing-api/models"
)

type V2UpdateRgMigration struct{}

var _ Migration = new(V2UpdateRgMigration)

func NewV2UpdateRgMigration() *V2UpdateRgMigration {
	return &V2UpdateRgMigration{}
}

func (v *V2UpdateRgMigration) Version() int {
	return 2
}

func (v *V2UpdateRgMigration) Run(sqlDB *db.SqlDB) error {
	// Check for duplicate router group names before creating the unique index
	var rgs []models.RouterGroupDB
	err := sqlDB.Client.Find(&rgs)
	if err != nil {
		return err
	}

	// Check for duplicates
	nameMap := make(map[string]int)
	for _, rg := range rgs {
		nameMap[rg.Name]++
	}

	for name, count := range nameMap {
		if count > 1 {
			return fmt.Errorf("cannot create unique index: router group name '%s' appears %d times", name, count)
		}
	}

	// Remove old index if it exists - using correct MySQL syntax with table name
	_ = sqlDB.Client.ExecWithError("DROP INDEX idx_rg_name ON router_groups")

	// Create unique index using raw SQL with column length specification
	err = sqlDB.Client.ExecWithError("CREATE UNIQUE INDEX idx_rg_name ON router_groups (name(191))")
	if err != nil {
		return err
	}

	return nil
}
