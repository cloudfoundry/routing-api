package migration

import (
	"os"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/routing-api/config"
	"code.cloudfoundry.org/routing-api/db"
	"github.com/jinzhu/gorm"
)

const MigrationKey = "routing-api-migration"

type MigrationData struct {
	MigrationKey   string `gorm:"primary_key"`
	CurrentVersion int
	TargetVersion  int
}

type Runner struct {
	etcdCfg *config.Etcd
	sqlCfg  *config.SqlDB
	logger  lager.Logger
}

func NewRunner(
	etcdCfg *config.Etcd,
	sqlCfg *config.SqlDB,
	logger lager.Logger,
) *Runner {
	return &Runner{
		etcdCfg: etcdCfg,
		sqlCfg:  sqlCfg,
		logger:  logger,
	}
}
func (r *Runner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	migrations := InitializeMigrations(r.etcdCfg, r.logger)

	if r.sqlCfg.IsValid() {

		err := RunMigrations(r.sqlCfg, migrations)
		if err != nil {
			r.logger.Error("migrations-failed", err)
			return err
		}
	}

	close(ready)

	select {
	case sig := <-signals:
		r.logger.Info("received signal %s", lager.Data{"signal": sig})
	}
	return nil
}

//go:generate counterfeiter -o fakes/fake_migration.go . Migration
type Migration interface {
	Run(*db.SqlDB) error
	Version() int
}

func InitializeMigrations(etcdCfg *config.Etcd, logger lager.Logger) []Migration {
	migrations := []Migration{}
	var migration Migration

	migration = NewV0InitMigration()
	migrations = append(migrations, migration)

	done := make(chan struct{})
	migration = NewV1EtcdMigration(etcdCfg, done, logger)
	migrations = append(migrations, migration)

	return migrations
}

func RunMigrations(sqlCfg *config.SqlDB, migrations []Migration) error {
	if len(migrations) == 0 {
		return nil
	}

	lastMigrationVersion := migrations[len(migrations)-1].Version()
	sqlDB, err := db.NewSqlDB(sqlCfg)
	if err != nil {
		return err
	}
	gormDB := sqlDB.Client.(*gorm.DB)

	defer gormDB.Close()
	gormDB.AutoMigrate(&MigrationData{})

	tx := gormDB.Begin()

	existingVersion := &MigrationData{}

	err = tx.Where("migration_key = ?", MigrationKey).First(existingVersion).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		tx.Rollback()
		return err
	}

	if err == gorm.ErrRecordNotFound {
		existingVersion = &MigrationData{
			MigrationKey:   MigrationKey,
			CurrentVersion: -1,
			TargetVersion:  lastMigrationVersion,
		}

		err := tx.Create(existingVersion).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	} else {
		if existingVersion.TargetVersion >= lastMigrationVersion {
			return tx.Commit().Error
		}

		existingVersion.TargetVersion = lastMigrationVersion
		err := tx.Save(existingVersion).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	err = tx.Commit().Error
	if err != nil {
		return err
	}

	currentVersion := existingVersion.CurrentVersion
	for _, m := range migrations {
		if m.Version() > currentVersion && m != nil {
			m.Run(sqlDB)
			currentVersion = m.Version()
			existingVersion.CurrentVersion = currentVersion
			err := gormDB.Save(existingVersion).Error
			if err != nil {
				return err
			}
		}
	}
	return nil
}
