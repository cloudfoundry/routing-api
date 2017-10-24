package testrunner

import (
	"database/sql"
	"errors"
	"fmt"
	"math/rand"

	. "github.com/onsi/ginkgo"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

type DbAllocator interface {
	Create() (string, error)
	Reset() error
	Delete() error
	ConnectionString() string
}

type mysqlAllocator struct {
	sqlDB      *sql.DB
	schemaName string
}

type postgresAllocator struct {
	sqlDB      *sql.DB
	schemaName string
}

func randSchemaName() string {
	return fmt.Sprintf("test%d%d", rand.Int(), GinkgoParallelNode())
}

func NewPostgresAllocator() DbAllocator {
	return &postgresAllocator{schemaName: randSchemaName()}
}
func NewMySQLAllocator() DbAllocator {
	return &mysqlAllocator{schemaName: randSchemaName()}
}

func (a *postgresAllocator) ConnectionString() string {
	return "postgres://postgres:@localhost/?sslmode=disable"
}

func (a *postgresAllocator) Create() (string, error) {
	var err error
	a.sqlDB, err = sql.Open("postgres", a.ConnectionString())
	if err != nil {
		return "", err
	}
	err = a.sqlDB.Ping()
	if err != nil {
		return "", err
	}

	for i := 0; i < 5; i++ {
		dbExists, err := a.sqlDB.Exec(fmt.Sprintf("SELECT * FROM pg_database WHERE datname='%s'", a.schemaName))
		rowsAffected, err := dbExists.RowsAffected()
		if err != nil {
			return "", err
		}
		if rowsAffected == 0 {
			_, err = a.sqlDB.Exec(fmt.Sprintf("CREATE DATABASE %s", a.schemaName))
			if err != nil {
				return "", err
			}
			return a.schemaName, nil
		} else {
			a.schemaName = randSchemaName()
		}
	}
	return "", errors.New("Failed to create unique database ")
}

func (a *postgresAllocator) Reset() error {
	_, err := a.sqlDB.Exec(fmt.Sprintf(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity
	WHERE datname = '%s'`, a.schemaName))
	_, err = a.sqlDB.Exec(fmt.Sprintf("DROP DATABASE %s", a.schemaName))
	if err != nil {
		return err
	}

	_, err = a.sqlDB.Exec(fmt.Sprintf("CREATE DATABASE %s", a.schemaName))
	return err
}

func (a *postgresAllocator) Delete() error {
	defer func() {
		_ = a.sqlDB.Close()
	}()
	_, err := a.sqlDB.Exec(fmt.Sprintf(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity
	WHERE datname = '%s'`, a.schemaName))
	if err != nil {
		return err
	}
	_, err = a.sqlDB.Exec(fmt.Sprintf("DROP DATABASE %s", a.schemaName))
	return err
}

func (a *mysqlAllocator) ConnectionString() string {
	return "root:password@/"
}

func (a *mysqlAllocator) Create() (string, error) {
	var err error
	a.sqlDB, err = sql.Open("mysql", a.ConnectionString())
	if err != nil {
		return "", err
	}
	err = a.sqlDB.Ping()
	if err != nil {
		return "", err
	}

	for i := 0; i < 5; i++ {
		dbExists, err := a.sqlDB.Exec(fmt.Sprintf("SHOW DATABASES LIKE '%s'", a.schemaName))
		rowsAffected, err := dbExists.RowsAffected()
		if err != nil {
			return "", err
		}
		if rowsAffected == 0 {
			_, err = a.sqlDB.Exec(fmt.Sprintf("CREATE DATABASE %s", a.schemaName))
			if err != nil {
				return "", err
			}
			return a.schemaName, nil
		} else {
			a.schemaName = randSchemaName()
		}
	}
	return "", errors.New("Failed to create unique database ")
}

func (a *mysqlAllocator) Reset() error {
	_, err := a.sqlDB.Exec(fmt.Sprintf("DROP DATABASE %s", a.schemaName))
	if err != nil {
		return err
	}

	_, err = a.sqlDB.Exec(fmt.Sprintf("CREATE DATABASE %s", a.schemaName))
	return err
}

func (a *mysqlAllocator) Delete() error {
	defer func() {
		_ = a.sqlDB.Close()
	}()
	_, err := a.sqlDB.Exec(fmt.Sprintf("DROP DATABASE %s", a.schemaName))
	return err
}
