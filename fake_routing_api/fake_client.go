// This file was generated by counterfeiter
package fake_routing_api

import (
	"sync"

	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api/db"
)

type FakeClient struct {
	SetTokenStub        func(string)
	setTokenMutex       sync.RWMutex
	setTokenArgsForCall []struct {
		arg1 string
	}
	UpsertRoutesStub        func([]db.Route) error
	upsertRoutesMutex       sync.RWMutex
	upsertRoutesArgsForCall []struct {
		arg1 []db.Route
	}
	upsertRoutesReturns struct {
		result1 error
	}
	RoutesStub        func() ([]db.Route, error)
	routesMutex       sync.RWMutex
	routesArgsForCall []struct{}
	routesReturns struct {
		result1 []db.Route
		result2 error
	}
	DeleteRoutesStub        func([]db.Route) error
	deleteRoutesMutex       sync.RWMutex
	deleteRoutesArgsForCall []struct {
		arg1 []db.Route
	}
	deleteRoutesReturns struct {
		result1 error
	}
	RouterGroupsStub        func() ([]db.RouterGroup, error)
	routerGroupsMutex       sync.RWMutex
	routerGroupsArgsForCall []struct{}
	routerGroupsReturns struct {
		result1 []db.RouterGroup
		result2 error
	}
	SubscribeToEventsStub        func() (routing_api.EventSource, error)
	subscribeToEventsMutex       sync.RWMutex
	subscribeToEventsArgsForCall []struct{}
	subscribeToEventsReturns struct {
		result1 routing_api.EventSource
		result2 error
	}
}

func (fake *FakeClient) SetToken(arg1 string) {
	fake.setTokenMutex.Lock()
	fake.setTokenArgsForCall = append(fake.setTokenArgsForCall, struct {
		arg1 string
	}{arg1})
	fake.setTokenMutex.Unlock()
	if fake.SetTokenStub != nil {
		fake.SetTokenStub(arg1)
	}
}

func (fake *FakeClient) SetTokenCallCount() int {
	fake.setTokenMutex.RLock()
	defer fake.setTokenMutex.RUnlock()
	return len(fake.setTokenArgsForCall)
}

func (fake *FakeClient) SetTokenArgsForCall(i int) string {
	fake.setTokenMutex.RLock()
	defer fake.setTokenMutex.RUnlock()
	return fake.setTokenArgsForCall[i].arg1
}

func (fake *FakeClient) UpsertRoutes(arg1 []db.Route) error {
	fake.upsertRoutesMutex.Lock()
	fake.upsertRoutesArgsForCall = append(fake.upsertRoutesArgsForCall, struct {
		arg1 []db.Route
	}{arg1})
	fake.upsertRoutesMutex.Unlock()
	if fake.UpsertRoutesStub != nil {
		return fake.UpsertRoutesStub(arg1)
	} else {
		return fake.upsertRoutesReturns.result1
	}
}

func (fake *FakeClient) UpsertRoutesCallCount() int {
	fake.upsertRoutesMutex.RLock()
	defer fake.upsertRoutesMutex.RUnlock()
	return len(fake.upsertRoutesArgsForCall)
}

func (fake *FakeClient) UpsertRoutesArgsForCall(i int) []db.Route {
	fake.upsertRoutesMutex.RLock()
	defer fake.upsertRoutesMutex.RUnlock()
	return fake.upsertRoutesArgsForCall[i].arg1
}

func (fake *FakeClient) UpsertRoutesReturns(result1 error) {
	fake.UpsertRoutesStub = nil
	fake.upsertRoutesReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeClient) Routes() ([]db.Route, error) {
	fake.routesMutex.Lock()
	fake.routesArgsForCall = append(fake.routesArgsForCall, struct{}{})
	fake.routesMutex.Unlock()
	if fake.RoutesStub != nil {
		return fake.RoutesStub()
	} else {
		return fake.routesReturns.result1, fake.routesReturns.result2
	}
}

func (fake *FakeClient) RoutesCallCount() int {
	fake.routesMutex.RLock()
	defer fake.routesMutex.RUnlock()
	return len(fake.routesArgsForCall)
}

func (fake *FakeClient) RoutesReturns(result1 []db.Route, result2 error) {
	fake.RoutesStub = nil
	fake.routesReturns = struct {
		result1 []db.Route
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) DeleteRoutes(arg1 []db.Route) error {
	fake.deleteRoutesMutex.Lock()
	fake.deleteRoutesArgsForCall = append(fake.deleteRoutesArgsForCall, struct {
		arg1 []db.Route
	}{arg1})
	fake.deleteRoutesMutex.Unlock()
	if fake.DeleteRoutesStub != nil {
		return fake.DeleteRoutesStub(arg1)
	} else {
		return fake.deleteRoutesReturns.result1
	}
}

func (fake *FakeClient) DeleteRoutesCallCount() int {
	fake.deleteRoutesMutex.RLock()
	defer fake.deleteRoutesMutex.RUnlock()
	return len(fake.deleteRoutesArgsForCall)
}

func (fake *FakeClient) DeleteRoutesArgsForCall(i int) []db.Route {
	fake.deleteRoutesMutex.RLock()
	defer fake.deleteRoutesMutex.RUnlock()
	return fake.deleteRoutesArgsForCall[i].arg1
}

func (fake *FakeClient) DeleteRoutesReturns(result1 error) {
	fake.DeleteRoutesStub = nil
	fake.deleteRoutesReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeClient) RouterGroups() ([]db.RouterGroup, error) {
	fake.routerGroupsMutex.Lock()
	fake.routerGroupsArgsForCall = append(fake.routerGroupsArgsForCall, struct{}{})
	fake.routerGroupsMutex.Unlock()
	if fake.RouterGroupsStub != nil {
		return fake.RouterGroupsStub()
	} else {
		return fake.routerGroupsReturns.result1, fake.routerGroupsReturns.result2
	}
}

func (fake *FakeClient) RouterGroupsCallCount() int {
	fake.routerGroupsMutex.RLock()
	defer fake.routerGroupsMutex.RUnlock()
	return len(fake.routerGroupsArgsForCall)
}

func (fake *FakeClient) RouterGroupsReturns(result1 []db.RouterGroup, result2 error) {
	fake.RouterGroupsStub = nil
	fake.routerGroupsReturns = struct {
		result1 []db.RouterGroup
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) SubscribeToEvents() (routing_api.EventSource, error) {
	fake.subscribeToEventsMutex.Lock()
	fake.subscribeToEventsArgsForCall = append(fake.subscribeToEventsArgsForCall, struct{}{})
	fake.subscribeToEventsMutex.Unlock()
	if fake.SubscribeToEventsStub != nil {
		return fake.SubscribeToEventsStub()
	} else {
		return fake.subscribeToEventsReturns.result1, fake.subscribeToEventsReturns.result2
	}
}

func (fake *FakeClient) SubscribeToEventsCallCount() int {
	fake.subscribeToEventsMutex.RLock()
	defer fake.subscribeToEventsMutex.RUnlock()
	return len(fake.subscribeToEventsArgsForCall)
}

func (fake *FakeClient) SubscribeToEventsReturns(result1 routing_api.EventSource, result2 error) {
	fake.SubscribeToEventsStub = nil
	fake.subscribeToEventsReturns = struct {
		result1 routing_api.EventSource
		result2 error
	}{result1, result2}
}

var _ routing_api.Client = new(FakeClient)
