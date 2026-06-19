module code.cloudfoundry.org/routing-api

go 1.26.4

replace github.com/cactus/go-statsd-client => github.com/cactus/go-statsd-client v2.0.2-0.20150911070441-6fa055a7b594+incompatible

require (
	code.cloudfoundry.org/cfhttp/v2 v2.82.0
	code.cloudfoundry.org/clock v1.75.0
	code.cloudfoundry.org/debugserver v0.102.0
	code.cloudfoundry.org/diego-logging-client v0.112.0
	code.cloudfoundry.org/eventhub v0.77.0
	code.cloudfoundry.org/go-loggregator/v9 v9.2.1
	code.cloudfoundry.org/lager/v3 v3.74.0
	code.cloudfoundry.org/locket v1.2.0
	code.cloudfoundry.org/tlsconfig v0.60.0
	github.com/cactus/go-statsd-client v3.2.1+incompatible
	github.com/cloudfoundry-community/go-uaa v0.4.0
	github.com/cloudfoundry/dropsonde v1.1.0
	github.com/go-sql-driver/mysql v1.10.0
	github.com/golang-jwt/jwt/v4 v4.5.2
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d
	github.com/onsi/ginkgo/v2 v2.31.0
	github.com/onsi/gomega v1.42.0
	github.com/tedsuo/ifrit v0.0.0-20260418191334-846868129986
	github.com/tedsuo/rata v1.0.0
	github.com/vito/go-sse v1.1.3
	golang.org/x/oauth2 v0.36.0
	google.golang.org/grpc v1.81.1
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/driver/mysql v1.6.0
	gorm.io/driver/postgres v1.6.0
	gorm.io/gorm v1.31.1
)

require (
	code.cloudfoundry.org/diego-db-helpers v0.3.0 // indirect
	code.cloudfoundry.org/durationjson v0.77.0 // indirect
	code.cloudfoundry.org/go-diodes v0.0.0-20260615142411-472d6bcdb3c6 // indirect
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/Masterminds/semver/v3 v3.5.0 // indirect
	github.com/bmizerany/pat v0.0.0-20210406213842-e4b6760bdd6f // indirect
	github.com/cloudfoundry/sonde-go v0.0.0-20260526083715-66f310f13c26 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20260604005048-7023385849c0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.10.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/openzipkin/zipkin-go v0.4.3 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/square/certstrap v1.3.0 // indirect
	go.step.sm/crypto v0.83.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.53.0 // indirect
	golang.org/x/mod v0.37.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	golang.org/x/tools v0.46.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260618152121-87f3d3e198d3 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
