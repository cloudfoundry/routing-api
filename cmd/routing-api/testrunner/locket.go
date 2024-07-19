package testrunner

import (
	"code.cloudfoundry.org/locket/cmd/locket/config"
	locketrunner "code.cloudfoundry.org/locket/cmd/locket/testrunner"
	"fmt"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	ginkgomon "github.com/tedsuo/ifrit/ginkgomon_v2"
	"os"
)

func StartLocket(
	locketPort uint16,
	locketBinPath string,
	databaseName string,
	caCert string,
) ifrit.Process {
	locketAddress := fmt.Sprintf("localhost:%d", locketPort)

	locketRunner := locketrunner.NewLocketRunner(locketBinPath, func(cfg *config.LocketConfig) {
		switch Database {
		case Postgres:
			cfg.DatabaseConnectionString = fmt.Sprintf(
				"user=%s password=%s host=%s dbname=%s",
				PostgresUsername,
				PostgresPassword,
				Host,
				databaseName,
			)
			cfg.DatabaseDriver = Postgres
		default:
			cfg.DatabaseConnectionString = fmt.Sprintf("%s:%s@/%s", MySQLUserName, MySQLPassword, databaseName)
			cfg.DatabaseDriver = MySQL
		}
		if caCert != "" {
			caFile, err := os.CreateTemp("", "")
			Expect(err).ToNot(HaveOccurred())
			Expect(os.WriteFile(caFile.Name(), []byte(caCert), 0400)).To(Succeed())
			cfg.SQLCACertFile = caFile.Name()
		}
		cfg.ListenAddress = locketAddress
	})

	return ginkgomon.Invoke(locketRunner)
}

func StopLocket(locketProcess ifrit.Process) {
	ginkgomon.Interrupt(locketProcess)
	locketProcess.Wait()
}
