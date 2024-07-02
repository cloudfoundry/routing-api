package fakes

import (
	"sync"

	"code.cloudfoundry.org/routing-api/metrics"
	"github.com/cactus/go-statsd-client/v5/statsd"
)

type FakePartialStatsdClient struct {
	GaugeDeltaStub        func(stat string, value int64, rate float32, tags ...statsd.Tag) error
	gaugeDeltaMutex       sync.RWMutex
	gaugeDeltaArgsForCall []struct {
		stat  string
		value int64
		rate  float32
		tags  []statsd.Tag
	}
	gaugeDeltaReturns struct {
		result1 error
	}
	GaugeStub        func(stat string, value int64, rate float32, tags ...statsd.Tag) error
	gaugeMutex       sync.RWMutex
	gaugeArgsForCall []struct {
		stat  string
		value int64
		rate  float32
		tags  []statsd.Tag
	}
	gaugeReturns struct {
		result1 error
	}
}

func (fake *FakePartialStatsdClient) GaugeDelta(stat string, value int64, rate float32, tags ...statsd.Tag) error {
	fake.gaugeDeltaMutex.Lock()
	fake.gaugeDeltaArgsForCall = append(fake.gaugeDeltaArgsForCall, struct {
		stat  string
		value int64
		rate  float32
		tags  []statsd.Tag
	}{stat, value, rate, tags})
	fake.gaugeDeltaMutex.Unlock()
	if fake.GaugeDeltaStub != nil {
		return fake.GaugeDeltaStub(stat, value, rate, tags...)
	} else {
		return fake.gaugeDeltaReturns.result1
	}
}

func (fake *FakePartialStatsdClient) GaugeDeltaCallCount() int {
	fake.gaugeDeltaMutex.RLock()
	defer fake.gaugeDeltaMutex.RUnlock()
	return len(fake.gaugeDeltaArgsForCall)
}

func (fake *FakePartialStatsdClient) GaugeDeltaArgsForCall(i int) (string, int64, float32) {
	fake.gaugeDeltaMutex.RLock()
	defer fake.gaugeDeltaMutex.RUnlock()
	return fake.gaugeDeltaArgsForCall[i].stat, fake.gaugeDeltaArgsForCall[i].value, fake.gaugeDeltaArgsForCall[i].rate
}

func (fake *FakePartialStatsdClient) GaugeDeltaReturns(result1 error) {
	fake.GaugeDeltaStub = nil
	fake.gaugeDeltaReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakePartialStatsdClient) Gauge(stat string, value int64, rate float32, tags ...statsd.Tag) error {
	fake.gaugeMutex.Lock()
	fake.gaugeArgsForCall = append(fake.gaugeArgsForCall, struct {
		stat  string
		value int64
		rate  float32
		tags  []statsd.Tag
	}{stat, value, rate, tags})
	fake.gaugeMutex.Unlock()
	if fake.GaugeStub != nil {
		return fake.GaugeStub(stat, value, rate, tags...)
	} else {
		return fake.gaugeReturns.result1
	}
}

func (fake *FakePartialStatsdClient) GaugeCallCount() int {
	fake.gaugeMutex.RLock()
	defer fake.gaugeMutex.RUnlock()
	return len(fake.gaugeArgsForCall)
}

func (fake *FakePartialStatsdClient) GaugeArgsForCall(i int) (string, int64, float32, []statsd.Tag) {
	fake.gaugeMutex.RLock()
	defer fake.gaugeMutex.RUnlock()
	return fake.gaugeArgsForCall[i].stat, fake.gaugeArgsForCall[i].value, fake.gaugeArgsForCall[i].rate, fake.gaugeArgsForCall[i].tags
}

func (fake *FakePartialStatsdClient) GaugeReturns(result1 error) {
	fake.GaugeStub = nil
	fake.gaugeReturns = struct {
		result1 error
	}{result1}
}

var _ metrics.PartialStatsdClient = new(FakePartialStatsdClient)
