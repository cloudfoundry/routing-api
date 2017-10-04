package admin_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"

	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/routing-api/admin"
	fake_db "code.cloudfoundry.org/routing-api/db/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/sigmon"
)

func tempUnixSocket() string {
	tmpfile, err := ioutil.TempFile("", "admin.sock")
	Expect(err).ToNot(HaveOccurred())
	defer Expect(os.Remove(tmpfile.Name())).To(Succeed())

	err = tmpfile.Close()
	Expect(err).ToNot(HaveOccurred())
	return tmpfile.Name()
}

var _ = Describe("AdminServer", func() {
	var (
		logger  *lagertest.TestLogger
		db      *fake_db.FakeDB
		socket  string
		process ifrit.Process
	)
	BeforeEach(func() {
		db = new(fake_db.FakeDB)
		logger = lagertest.NewTestLogger("routing-api-test")
		socket = tempUnixSocket()
		server := admin.NewServer(socket, db, logger)
		process = ifrit.Invoke(sigmon.New(server))
		Eventually(process.Ready()).Should(BeClosed())
	})

	AfterEach(func() {
		process.Signal(os.Interrupt)
		Eventually(process.Wait()).Should(Receive())
		_, err := os.Stat(socket)
		Expect(os.IsNotExist(err)).To(BeTrue())
	})

	DescribeTable("testing the admin endpoint",
		func(endpoint string, callCountFn func(db *fake_db.FakeDB) int) {
			req, _ := http.NewRequest("PUT", fmt.Sprintf("http://whatever/%s", endpoint), nil)
			tr := &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", socket)
				},
			}
			client := http.Client{Transport: tr}

			resp, err := client.Do(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(callCountFn(db)).To(Equal(1))
		},
		Entry(`"lock_router_group_reads"`, "lock_router_group_reads", func(db *fake_db.FakeDB) int {
			return db.LockRouterGroupReadsCallCount()
		}),
		Entry(`"unlock_router_group_reads"`, "unlock_router_group_reads", func(db *fake_db.FakeDB) int {
			return db.UnlockRouterGroupReadsCallCount()
		}),
		Entry(`"lock_router_group_writes"`, "lock_router_group_writes", func(db *fake_db.FakeDB) int {
			return db.LockRouterGroupWritesCallCount()
		}),
		Entry(`"unlock_router_group_writes"`, "unlock_router_group_writes", func(db *fake_db.FakeDB) int {
			return db.UnlockRouterGroupWritesCallCount()
		}),
	)
})
