package db

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"code.cloudfoundry.org/routing-api/config"
	"github.com/go-sql-driver/mysql"
)

//go:generate counterfeiter -o fakes/fake_mysql_adapter.go --fake-name MySQLAdapter . mySQLAdapter
type mySQLAdapter interface {
	RegisterTLSConfig(key string, config *tls.Config) error
}

type MySQLConnectionStringBuilder struct {
	MySQLAdapter mySQLAdapter
}

func (m *MySQLConnectionStringBuilder) Build(cfg *config.SqlDB) (string, error) {
	rootCA := x509.NewCertPool()
	queryString := "?parseTime=true"
	if cfg.SkipSSLValidation {
		tlsConfig := tls.Config{}
		tlsConfig.InsecureSkipVerify = cfg.SkipSSLValidation
		configKey := "dbTLSSkipVerify"
		err := mysql.RegisterTLSConfig(configKey, &tlsConfig)
		if err != nil {
			return "", err
		}
		queryString = fmt.Sprintf("%s&tls=%s", queryString, configKey)

	} else if cfg.CACert != "" {
		tlsConfig := tls.Config{}
		rootCA.AppendCertsFromPEM([]byte(cfg.CACert))
		tlsConfig.ServerName = cfg.Host
		tlsConfig.RootCAs = rootCA
		configKey := "dbTLSCertVerify"
		err := mysql.RegisterTLSConfig(configKey, &tlsConfig)
		if err != nil {
			return "", err
		}
		queryString = fmt.Sprintf("%s&tls=%s", queryString, configKey)
	}
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s%s",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Schema,
		queryString,
	), nil
}
