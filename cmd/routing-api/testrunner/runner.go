package testrunner

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"time"

	"code.cloudfoundry.org/cf-tcp-router/utils"
	"code.cloudfoundry.org/routing-api/config"
	"code.cloudfoundry.org/routing-api/test_helpers"
	"github.com/tedsuo/ifrit/ginkgomon"
	yaml "gopkg.in/yaml.v2"
)

var dbEnv = os.Getenv("DB")

type Args struct {
	ConfigPath string
	DevMode    bool
	IP         string
}

func (args Args) ArgSlice() []string {
	return []string{
		"-ip", args.IP,
		"-config", args.ConfigPath,
		"-logLevel=debug",
		"-devMode=" + strconv.FormatBool(args.DevMode),
	}
}

func NewDbAllocator() DbAllocator {
	var dbAllocator DbAllocator
	switch dbEnv {
	case "postgres":
		dbAllocator = NewPostgresAllocator()
	default:
		dbAllocator = NewMySQLAllocator()
	}
	return dbAllocator
}

func NewRoutingAPIArgs(ip string, port uint16, dbId, dbCACert, consulUrl string) (Args, error) {
	configPath, err := createConfig(dbId, dbCACert, consulUrl)
	if err != nil {
		return Args{}, err
	}
	return Args{
		IP:         ip,
		ConfigPath: configPath,
		DevMode:    true,
	}, nil
}

func New(binPath string, args Args) *ginkgomon.Runner {
	cmd := exec.Command(binPath, args.ArgSlice()...)
	return ginkgomon.New(ginkgomon.Config{
		Name:              "routing-api",
		Command:           cmd,
		StartCheck:        "routing-api.started",
		StartCheckTimeout: 30 * time.Second,
	})
}

func createConfig(dbId, dbCACert, consulUrl string) (string, error) {
	var configBytes []byte
	configFile, err := ioutil.TempFile("", "routing-api-config")
	if err != nil {
		return "", err
	}
	configFilePath := configFile.Name()
	adminPort := test_helpers.NextAvailPort()

	type SqlConfig struct {
		SqlDB config.SqlDB `yaml:"sqldb"`
	}
	switch dbEnv {
	case "postgres":

		dbConfig := config.SqlDB{
			Username: "postgres",
			Password: "",
			Schema:   dbId,
			Port:     5432,
			Host:     "localhost",
			Type:     "postgres",
			CACert:   dbCACert,
		}
		config := SqlConfig{
			SqlDB: dbConfig,
		}
		postgresConfig, err := yaml.Marshal(&config)
		if err != nil {
			return "", err
		}

		postgresConfigStr := `log_guid: "my_logs"
uaa_verification_key: "-----BEGIN PUBLIC KEY-----

      MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDHFr+KICms+tuT1OXJwhCUmR2d

      KVy7psa8xzElSyzqx7oJyfJ1JZyOzToj9T5SfTIq396agbHJWVfYphNahvZ/7uMX

      qHxf+ZH9BL1gk9Y6kCnbM5R60gfwjyW1/dQPjOzn9N394zd2FJoFHwdq9Qs0wBug

      spULZVNRxq7veq/fzwIDAQAB

      -----END PUBLIC KEY-----"

uuid: "routing-api-uuid"
debug_address: "1.2.3.4:1234"
metron_config:
  address: "1.2.3.4"
  port: "4567"
metrics_reporting_interval: "500ms"
statsd_endpoint: "localhost:8125"
statsd_client_flush_interval: "10ms"
system_domain: "example.com"
admin_port: %d
router_groups:
- name: "default-tcp"
  type: "tcp"
  reservable_ports: "1024-65535"
consul_cluster:
  servers: "%s"
  retry_interval: 50ms
%s`
		configBytes = []byte(fmt.Sprintf(postgresConfigStr, adminPort, consulUrl, string(postgresConfig)))
	default:

		dbConfig := config.SqlDB{
			Username: "root",
			Password: "password",
			Schema:   dbId,
			Port:     3306,
			Host:     "localhost",
			Type:     "mysql",
			CACert:   dbCACert,
		}
		config := SqlConfig{
			SqlDB: dbConfig,
		}
		mysqlConfig, err := yaml.Marshal(&config)
		if err != nil {
			return "", err
		}
		mysqlConfigStr := `log_guid: "my_logs"
uaa_verification_key: "-----BEGIN PUBLIC KEY-----

      MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDHFr+KICms+tuT1OXJwhCUmR2d

      KVy7psa8xzElSyzqx7oJyfJ1JZyOzToj9T5SfTIq396agbHJWVfYphNahvZ/7uMX

      qHxf+ZH9BL1gk9Y6kCnbM5R60gfwjyW1/dQPjOzn9N394zd2FJoFHwdq9Qs0wBug

      spULZVNRxq7veq/fzwIDAQAB

      -----END PUBLIC KEY-----"

uuid: "routing-api-uuid"
debug_address: "1.2.3.4:1234"
metron_config:
  address: "1.2.3.4"
  port: "4567"
metrics_reporting_interval: "500ms"
statsd_endpoint: "localhost:8125"
statsd_client_flush_interval: "10ms"
system_domain: "example.com"
admin_port: %d
router_groups:
- name: "default-tcp"
  type: "tcp"
  reservable_ports: "1024-65535"
consul_cluster:
  servers: "%s"
  retry_interval: 50ms
%s`
		configBytes = []byte(fmt.Sprintf(mysqlConfigStr, adminPort, consulUrl, string(mysqlConfig)))
	}

	err = utils.WriteToFile(configBytes, configFilePath)
	return configFilePath, err
}
