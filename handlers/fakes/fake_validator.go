// Code generated by counterfeiter. DO NOT EDIT.
package fakes

import (
	"sync"

	routing_api "code.cloudfoundry.org/routing-api"
	"code.cloudfoundry.org/routing-api/handlers"
	"code.cloudfoundry.org/routing-api/models"
)

type FakeRouteValidator struct {
	ValidateCreateStub        func([]models.Route, int) *routing_api.Error
	validateCreateMutex       sync.RWMutex
	validateCreateArgsForCall []struct {
		arg1 []models.Route
		arg2 int
	}
	validateCreateReturns struct {
		result1 *routing_api.Error
	}
	validateCreateReturnsOnCall map[int]struct {
		result1 *routing_api.Error
	}
	ValidateCreateTcpRouteMappingStub        func([]models.TcpRouteMapping, models.RouterGroups, int) *routing_api.Error
	validateCreateTcpRouteMappingMutex       sync.RWMutex
	validateCreateTcpRouteMappingArgsForCall []struct {
		arg1 []models.TcpRouteMapping
		arg2 models.RouterGroups
		arg3 int
	}
	validateCreateTcpRouteMappingReturns struct {
		result1 *routing_api.Error
	}
	validateCreateTcpRouteMappingReturnsOnCall map[int]struct {
		result1 *routing_api.Error
	}
	ValidateDeleteStub        func([]models.Route) *routing_api.Error
	validateDeleteMutex       sync.RWMutex
	validateDeleteArgsForCall []struct {
		arg1 []models.Route
	}
	validateDeleteReturns struct {
		result1 *routing_api.Error
	}
	validateDeleteReturnsOnCall map[int]struct {
		result1 *routing_api.Error
	}
	ValidateDeleteTcpRouteMappingStub        func([]models.TcpRouteMapping) *routing_api.Error
	validateDeleteTcpRouteMappingMutex       sync.RWMutex
	validateDeleteTcpRouteMappingArgsForCall []struct {
		arg1 []models.TcpRouteMapping
	}
	validateDeleteTcpRouteMappingReturns struct {
		result1 *routing_api.Error
	}
	validateDeleteTcpRouteMappingReturnsOnCall map[int]struct {
		result1 *routing_api.Error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeRouteValidator) ValidateCreate(arg1 []models.Route, arg2 int) *routing_api.Error {
	var arg1Copy []models.Route
	if arg1 != nil {
		arg1Copy = make([]models.Route, len(arg1))
		copy(arg1Copy, arg1)
	}
	fake.validateCreateMutex.Lock()
	ret, specificReturn := fake.validateCreateReturnsOnCall[len(fake.validateCreateArgsForCall)]
	fake.validateCreateArgsForCall = append(fake.validateCreateArgsForCall, struct {
		arg1 []models.Route
		arg2 int
	}{arg1Copy, arg2})
	stub := fake.ValidateCreateStub
	fakeReturns := fake.validateCreateReturns
	fake.recordInvocation("ValidateCreate", []interface{}{arg1Copy, arg2})
	fake.validateCreateMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeRouteValidator) ValidateCreateCallCount() int {
	fake.validateCreateMutex.RLock()
	defer fake.validateCreateMutex.RUnlock()
	return len(fake.validateCreateArgsForCall)
}

func (fake *FakeRouteValidator) ValidateCreateCalls(stub func([]models.Route, int) *routing_api.Error) {
	fake.validateCreateMutex.Lock()
	defer fake.validateCreateMutex.Unlock()
	fake.ValidateCreateStub = stub
}

func (fake *FakeRouteValidator) ValidateCreateArgsForCall(i int) ([]models.Route, int) {
	fake.validateCreateMutex.RLock()
	defer fake.validateCreateMutex.RUnlock()
	argsForCall := fake.validateCreateArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeRouteValidator) ValidateCreateReturns(result1 *routing_api.Error) {
	fake.validateCreateMutex.Lock()
	defer fake.validateCreateMutex.Unlock()
	fake.ValidateCreateStub = nil
	fake.validateCreateReturns = struct {
		result1 *routing_api.Error
	}{result1}
}

func (fake *FakeRouteValidator) ValidateCreateReturnsOnCall(i int, result1 *routing_api.Error) {
	fake.validateCreateMutex.Lock()
	defer fake.validateCreateMutex.Unlock()
	fake.ValidateCreateStub = nil
	if fake.validateCreateReturnsOnCall == nil {
		fake.validateCreateReturnsOnCall = make(map[int]struct {
			result1 *routing_api.Error
		})
	}
	fake.validateCreateReturnsOnCall[i] = struct {
		result1 *routing_api.Error
	}{result1}
}

func (fake *FakeRouteValidator) ValidateCreateTcpRouteMapping(arg1 []models.TcpRouteMapping, arg2 models.RouterGroups, arg3 int) *routing_api.Error {
	var arg1Copy []models.TcpRouteMapping
	if arg1 != nil {
		arg1Copy = make([]models.TcpRouteMapping, len(arg1))
		copy(arg1Copy, arg1)
	}
	fake.validateCreateTcpRouteMappingMutex.Lock()
	ret, specificReturn := fake.validateCreateTcpRouteMappingReturnsOnCall[len(fake.validateCreateTcpRouteMappingArgsForCall)]
	fake.validateCreateTcpRouteMappingArgsForCall = append(fake.validateCreateTcpRouteMappingArgsForCall, struct {
		arg1 []models.TcpRouteMapping
		arg2 models.RouterGroups
		arg3 int
	}{arg1Copy, arg2, arg3})
	stub := fake.ValidateCreateTcpRouteMappingStub
	fakeReturns := fake.validateCreateTcpRouteMappingReturns
	fake.recordInvocation("ValidateCreateTcpRouteMapping", []interface{}{arg1Copy, arg2, arg3})
	fake.validateCreateTcpRouteMappingMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeRouteValidator) ValidateCreateTcpRouteMappingCallCount() int {
	fake.validateCreateTcpRouteMappingMutex.RLock()
	defer fake.validateCreateTcpRouteMappingMutex.RUnlock()
	return len(fake.validateCreateTcpRouteMappingArgsForCall)
}

func (fake *FakeRouteValidator) ValidateCreateTcpRouteMappingCalls(stub func([]models.TcpRouteMapping, models.RouterGroups, int) *routing_api.Error) {
	fake.validateCreateTcpRouteMappingMutex.Lock()
	defer fake.validateCreateTcpRouteMappingMutex.Unlock()
	fake.ValidateCreateTcpRouteMappingStub = stub
}

func (fake *FakeRouteValidator) ValidateCreateTcpRouteMappingArgsForCall(i int) ([]models.TcpRouteMapping, models.RouterGroups, int) {
	fake.validateCreateTcpRouteMappingMutex.RLock()
	defer fake.validateCreateTcpRouteMappingMutex.RUnlock()
	argsForCall := fake.validateCreateTcpRouteMappingArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeRouteValidator) ValidateCreateTcpRouteMappingReturns(result1 *routing_api.Error) {
	fake.validateCreateTcpRouteMappingMutex.Lock()
	defer fake.validateCreateTcpRouteMappingMutex.Unlock()
	fake.ValidateCreateTcpRouteMappingStub = nil
	fake.validateCreateTcpRouteMappingReturns = struct {
		result1 *routing_api.Error
	}{result1}
}

func (fake *FakeRouteValidator) ValidateCreateTcpRouteMappingReturnsOnCall(i int, result1 *routing_api.Error) {
	fake.validateCreateTcpRouteMappingMutex.Lock()
	defer fake.validateCreateTcpRouteMappingMutex.Unlock()
	fake.ValidateCreateTcpRouteMappingStub = nil
	if fake.validateCreateTcpRouteMappingReturnsOnCall == nil {
		fake.validateCreateTcpRouteMappingReturnsOnCall = make(map[int]struct {
			result1 *routing_api.Error
		})
	}
	fake.validateCreateTcpRouteMappingReturnsOnCall[i] = struct {
		result1 *routing_api.Error
	}{result1}
}

func (fake *FakeRouteValidator) ValidateDelete(arg1 []models.Route) *routing_api.Error {
	var arg1Copy []models.Route
	if arg1 != nil {
		arg1Copy = make([]models.Route, len(arg1))
		copy(arg1Copy, arg1)
	}
	fake.validateDeleteMutex.Lock()
	ret, specificReturn := fake.validateDeleteReturnsOnCall[len(fake.validateDeleteArgsForCall)]
	fake.validateDeleteArgsForCall = append(fake.validateDeleteArgsForCall, struct {
		arg1 []models.Route
	}{arg1Copy})
	stub := fake.ValidateDeleteStub
	fakeReturns := fake.validateDeleteReturns
	fake.recordInvocation("ValidateDelete", []interface{}{arg1Copy})
	fake.validateDeleteMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeRouteValidator) ValidateDeleteCallCount() int {
	fake.validateDeleteMutex.RLock()
	defer fake.validateDeleteMutex.RUnlock()
	return len(fake.validateDeleteArgsForCall)
}

func (fake *FakeRouteValidator) ValidateDeleteCalls(stub func([]models.Route) *routing_api.Error) {
	fake.validateDeleteMutex.Lock()
	defer fake.validateDeleteMutex.Unlock()
	fake.ValidateDeleteStub = stub
}

func (fake *FakeRouteValidator) ValidateDeleteArgsForCall(i int) []models.Route {
	fake.validateDeleteMutex.RLock()
	defer fake.validateDeleteMutex.RUnlock()
	argsForCall := fake.validateDeleteArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeRouteValidator) ValidateDeleteReturns(result1 *routing_api.Error) {
	fake.validateDeleteMutex.Lock()
	defer fake.validateDeleteMutex.Unlock()
	fake.ValidateDeleteStub = nil
	fake.validateDeleteReturns = struct {
		result1 *routing_api.Error
	}{result1}
}

func (fake *FakeRouteValidator) ValidateDeleteReturnsOnCall(i int, result1 *routing_api.Error) {
	fake.validateDeleteMutex.Lock()
	defer fake.validateDeleteMutex.Unlock()
	fake.ValidateDeleteStub = nil
	if fake.validateDeleteReturnsOnCall == nil {
		fake.validateDeleteReturnsOnCall = make(map[int]struct {
			result1 *routing_api.Error
		})
	}
	fake.validateDeleteReturnsOnCall[i] = struct {
		result1 *routing_api.Error
	}{result1}
}

func (fake *FakeRouteValidator) ValidateDeleteTcpRouteMapping(arg1 []models.TcpRouteMapping) *routing_api.Error {
	var arg1Copy []models.TcpRouteMapping
	if arg1 != nil {
		arg1Copy = make([]models.TcpRouteMapping, len(arg1))
		copy(arg1Copy, arg1)
	}
	fake.validateDeleteTcpRouteMappingMutex.Lock()
	ret, specificReturn := fake.validateDeleteTcpRouteMappingReturnsOnCall[len(fake.validateDeleteTcpRouteMappingArgsForCall)]
	fake.validateDeleteTcpRouteMappingArgsForCall = append(fake.validateDeleteTcpRouteMappingArgsForCall, struct {
		arg1 []models.TcpRouteMapping
	}{arg1Copy})
	stub := fake.ValidateDeleteTcpRouteMappingStub
	fakeReturns := fake.validateDeleteTcpRouteMappingReturns
	fake.recordInvocation("ValidateDeleteTcpRouteMapping", []interface{}{arg1Copy})
	fake.validateDeleteTcpRouteMappingMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeRouteValidator) ValidateDeleteTcpRouteMappingCallCount() int {
	fake.validateDeleteTcpRouteMappingMutex.RLock()
	defer fake.validateDeleteTcpRouteMappingMutex.RUnlock()
	return len(fake.validateDeleteTcpRouteMappingArgsForCall)
}

func (fake *FakeRouteValidator) ValidateDeleteTcpRouteMappingCalls(stub func([]models.TcpRouteMapping) *routing_api.Error) {
	fake.validateDeleteTcpRouteMappingMutex.Lock()
	defer fake.validateDeleteTcpRouteMappingMutex.Unlock()
	fake.ValidateDeleteTcpRouteMappingStub = stub
}

func (fake *FakeRouteValidator) ValidateDeleteTcpRouteMappingArgsForCall(i int) []models.TcpRouteMapping {
	fake.validateDeleteTcpRouteMappingMutex.RLock()
	defer fake.validateDeleteTcpRouteMappingMutex.RUnlock()
	argsForCall := fake.validateDeleteTcpRouteMappingArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeRouteValidator) ValidateDeleteTcpRouteMappingReturns(result1 *routing_api.Error) {
	fake.validateDeleteTcpRouteMappingMutex.Lock()
	defer fake.validateDeleteTcpRouteMappingMutex.Unlock()
	fake.ValidateDeleteTcpRouteMappingStub = nil
	fake.validateDeleteTcpRouteMappingReturns = struct {
		result1 *routing_api.Error
	}{result1}
}

func (fake *FakeRouteValidator) ValidateDeleteTcpRouteMappingReturnsOnCall(i int, result1 *routing_api.Error) {
	fake.validateDeleteTcpRouteMappingMutex.Lock()
	defer fake.validateDeleteTcpRouteMappingMutex.Unlock()
	fake.ValidateDeleteTcpRouteMappingStub = nil
	if fake.validateDeleteTcpRouteMappingReturnsOnCall == nil {
		fake.validateDeleteTcpRouteMappingReturnsOnCall = make(map[int]struct {
			result1 *routing_api.Error
		})
	}
	fake.validateDeleteTcpRouteMappingReturnsOnCall[i] = struct {
		result1 *routing_api.Error
	}{result1}
}

func (fake *FakeRouteValidator) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.validateCreateMutex.RLock()
	defer fake.validateCreateMutex.RUnlock()
	fake.validateCreateTcpRouteMappingMutex.RLock()
	defer fake.validateCreateTcpRouteMappingMutex.RUnlock()
	fake.validateDeleteMutex.RLock()
	defer fake.validateDeleteMutex.RUnlock()
	fake.validateDeleteTcpRouteMappingMutex.RLock()
	defer fake.validateDeleteTcpRouteMappingMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeRouteValidator) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ handlers.RouteValidator = new(FakeRouteValidator)
