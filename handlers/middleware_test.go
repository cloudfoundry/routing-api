package handlers_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	cf_lager "github.com/cloudfoundry-incubator/cf-lager"
	"github.com/cloudfoundry-incubator/routing-api/handlers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager"
	. "github.com/pivotal-golang/lager/chug"
)

var _ = Describe("Middleware", func() {
	var (
		ts           *httptest.Server
		dummyHandler http.HandlerFunc
		stream       chan Entry
		pipeReader   *io.PipeReader
		pipeWriter   *io.PipeWriter
	)

	BeforeEach(func() {

		// logger
		logger := cf_lager.New("dummy-api")

		// dummy handler
		dummyHandler = func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Dummy handler")
		}

		// wrap dummy handler in logwrap
		dummyHandler = handlers.LogWrap(dummyHandler, logger)

		// test server
		ts = httptest.NewServer(dummyHandler)

		pipeReader, pipeWriter = io.Pipe()
		logger.RegisterSink(lager.NewWriterSink(pipeWriter, lager.DEBUG))
		stream = make(chan Entry, 100)
		go Chug(pipeReader, stream)
	})

	AfterEach(func() {
		ts.Close()
	})

	It("doesn't output the authorization information", func() {

		client := &http.Client{
			CheckRedirect: nil,
		}

		req, err := http.NewRequest("GET", ts.URL, nil)
		req.Header.Add("Authorization", "this-is-a-secret")
		resp, err := client.Do(req)

		// res, err := http.Get(ts.URL)
		Expect(err).NotTo(HaveOccurred())

		output, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		Expect(err).NotTo(HaveOccurred())

		Expect(output).To(ContainSubstring("Dummy handler"))

		entry := <-stream

		fmt.Printf("log data entry: %#v\n", entry.Log.Data)
		Expect(entry.Log).ToNot(BeNil())
		Expect(entry.Log.Data).ToNot(BeNil())
		Expect(entry.Log.Data["request-headers"]).ToNot(BeNil())
		Expect(entry.Log.Data["request-headers"]).ToNot(HaveKey("Authorization"))

	})
})
