package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"

	"code.cloudfoundry.org/locket"
	"code.cloudfoundry.org/routing-api/models"
)

type MetronConfig struct {
	Address string
	Port    string
}

type OAuthConfig struct {
	TokenEndpoint     string `yaml:"token_endpoint"`
	Port              int    `yaml:"port"`
	SkipSSLValidation bool   `yaml:"-"`
	ClientName        string `yaml:"client_name"`
	ClientSecret      string `yaml:"client_secret"`
	CACerts           string `yaml:"ca_certs"`
}

type SqlDB struct {
	Host                   string `yaml:"host"`
	Port                   int    `yaml:"port"`
	Schema                 string `yaml:"schema"`
	Type                   string `yaml:"type"`
	Username               string `yaml:"username"`
	Password               string `yaml:"password"`
	CACert                 string `yaml:"ca_cert"`
	SkipSSLValidation      bool   `yaml:"-"`
	SkipHostnameValidation bool   `yaml:"skip_hostname_validation"`
}

type ConsulCluster struct {
	Servers       string        `yaml:"servers"`
	LockTTL       time.Duration `yaml:"lock_ttl"`
	RetryInterval time.Duration `yaml:"retry_interval"`
}

type APIConfig struct {
	ListenPort int `yaml:"listen_port"`
}

type Config struct {
	API                             APIConfig                 `yaml:"api"`
	AdminPort                       int                       `yaml:"admin_port"`
	DebugAddress                    string                    `yaml:"debug_address"`
	LogGuid                         string                    `yaml:"log_guid"`
	MetronConfig                    MetronConfig              `yaml:"metron_config"`
	MaxTTL                          time.Duration             `yaml:"max_ttl"`
	SystemDomain                    string                    `yaml:"system_domain"`
	MetricsReportingIntervalString  string                    `yaml:"metrics_reporting_interval"`
	MetricsReportingInterval        time.Duration             `yaml:"-"`
	StatsdEndpoint                  string                    `yaml:"statsd_endpoint"`
	StatsdClientFlushIntervalString string                    `yaml:"statsd_client_flush_interval"`
	StatsdClientFlushInterval       time.Duration             `yaml:"-"`
	OAuth                           OAuthConfig               `yaml:"oauth"`
	RouterGroups                    models.RouterGroups       `yaml:"router_groups"`
	SqlDB                           SqlDB                     `yaml:"sqldb"`
	ConsulCluster                   ConsulCluster             `yaml:"consul_cluster"`
	SkipConsulLock                  bool                      `yaml:"skip_consul_lock"`
	Locket                          locket.ClientLocketConfig `yaml:"locket"`
	UUID                            string                    `yaml:"uuid"`
	SkipSSLValidation               bool                      `yaml:"skip_ssl_validation"`
}

func NewConfigFromFile(configFile string, authDisabled bool) (Config, error) {
	c, err := ioutil.ReadFile(configFile)
	if err != nil {
		return Config{}, err
	}
	return NewConfigFromBytes(c, authDisabled)
}

func NewConfigFromBytes(bytes []byte, authDisabled bool) (Config, error) {
	config := Config{}
	err := yaml.Unmarshal(bytes, &config)
	if err != nil {
		return config, err
	}

	err = config.validate(authDisabled)
	if err != nil {
		return config, err
	}

	err = config.process()
	if err != nil {
		return config, err
	}

	return config, nil
}

func (cfg *Config) validate(authDisabled bool) error {
	if cfg.SystemDomain == "" {
		return errors.New("No system_domain specified")
	}

	if cfg.LogGuid == "" {
		return errors.New("No log_guid specified")
	}

	if !authDisabled && cfg.OAuth.TokenEndpoint == "" {
		return errors.New("No token endpoint specified")
	}

	if !authDisabled && cfg.OAuth.TokenEndpoint != "" && cfg.OAuth.Port == -1 {
		return errors.New("Routing API requires TLS enabled to get OAuth token")
	}

	if cfg.UUID == "" {
		return errors.New("No UUID is specified")
	}

	if err := validatePort(cfg.AdminPort); err != nil {
		return fmt.Errorf("invalid admin port: %s", err)
	}

	if err := validatePort(cfg.API.ListenPort); err != nil {
		return fmt.Errorf("invalid API listen port: %s", err)
	}

	if err := cfg.RouterGroups.Validate(); err != nil {
		return err
	}

	return nil
}

func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port number is invalid: %d (1-65535)", port)
	}

	return nil
}

func (cfg *Config) process() error {
	if cfg.ConsulCluster.LockTTL == 0 {
		cfg.ConsulCluster.LockTTL = locket.DefaultSessionTTL
	}

	if cfg.ConsulCluster.RetryInterval == 0 {
		cfg.ConsulCluster.RetryInterval = locket.RetryInterval
	}

	cfg.SqlDB.SkipSSLValidation = cfg.SkipSSLValidation
	cfg.OAuth.SkipSSLValidation = cfg.SkipSSLValidation

	metricsReportingInterval, err := time.ParseDuration(cfg.MetricsReportingIntervalString)
	if err != nil {
		return err
	}
	cfg.MetricsReportingInterval = metricsReportingInterval

	statsdClientFlushInterval, err := time.ParseDuration(cfg.StatsdClientFlushIntervalString)
	if err != nil {
		return err
	}
	cfg.StatsdClientFlushInterval = statsdClientFlushInterval

	if cfg.MaxTTL == 0 {
		cfg.MaxTTL = 2 * time.Minute
	}

	return nil
}
