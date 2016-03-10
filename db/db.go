package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/cloudfoundry-incubator/routing-api/models"
	"github.com/cloudfoundry/gunk/workpool"
	"github.com/cloudfoundry/storeadapter"
	"github.com/cloudfoundry/storeadapter/etcdstoreadapter"
)

//go:generate counterfeiter -o fakes/fake_db.go . DB
type DB interface {
	ReadRoutes() ([]models.Route, error)
	SaveRoute(route models.Route) error
	DeleteRoute(route models.Route) error

	ReadTcpRouteMappings() ([]models.TcpRouteMapping, error)
	SaveTcpRouteMapping(tcpMapping models.TcpRouteMapping) error
	DeleteTcpRouteMapping(tcpMapping models.TcpRouteMapping) error

	ReadRouterGroups() (models.RouterGroups, error)
	SaveRouterGroup(routerGroup models.RouterGroup) error

	Connect() error
	Disconnect() error
	WatchRouteChanges(filter string) (<-chan storeadapter.WatchEvent, chan<- bool, <-chan error)
}

const (
	TCP_MAPPING_BASE_KEY  string = "/v1/tcp_routes/router_groups"
	HTTP_ROUTE_BASE_KEY   string = "/routes"
	ROUTER_GROUP_BASE_KEY string = "/v1/router_groups"
)

type etcd struct {
	storeAdapter *etcdstoreadapter.ETCDStoreAdapter
}

func NewETCD(nodeURLs []string, maxWorkers uint) (*etcd, error) {
	workpool, err := workpool.NewWorkPool(int(maxWorkers))
	if err != nil {
		return nil, err
	}

	storeAdapter, err := etcdstoreadapter.New(&etcdstoreadapter.ETCDOptions{ClusterUrls: nodeURLs}, workpool)
	if err != nil {
		return nil, err
	}
	return &etcd{
		storeAdapter: storeAdapter,
	}, nil
}

func (e *etcd) Connect() error {
	return e.storeAdapter.Connect()
}

func (e *etcd) Disconnect() error {
	return e.storeAdapter.Disconnect()
}

func (e *etcd) ReadRoutes() ([]models.Route, error) {
	routes, err := e.storeAdapter.ListRecursively(HTTP_ROUTE_BASE_KEY)
	if err != nil {
		return []models.Route{}, nil
	}

	listRoutes := []models.Route{}
	for _, node := range routes.ChildNodes {
		route := models.Route{}
		json.Unmarshal([]byte(node.Value), &route)
		listRoutes = append(listRoutes, route)
	}
	return listRoutes, nil
}

func (e *etcd) SaveRoute(route models.Route) error {
	key := generateHttpRouteKey(route)
	routeJSON, _ := json.Marshal(route)
	node := storeadapter.StoreNode{
		Key:   key,
		Value: routeJSON,
		TTL:   uint64(route.TTL),
	}

	return e.storeAdapter.SetMulti([]storeadapter.StoreNode{node})
}

func (e *etcd) DeleteRoute(route models.Route) error {
	key := generateHttpRouteKey(route)
	err := e.storeAdapter.Delete(key)
	if err != nil && err.Error() == "the requested key could not be found" {
		err = DBError{Type: KeyNotFound, Message: "The specified route could not be found."}
	}
	return err
}

func (e *etcd) WatchRouteChanges(filter string) (<-chan storeadapter.WatchEvent, chan<- bool, <-chan error) {
	return e.storeAdapter.Watch(filter)
}

func (e *etcd) SaveRouterGroup(routerGroup models.RouterGroup) error {
	if routerGroup.Guid == "" {
		return errors.New("Invalid router group: missing guid")
	}

	// fetch router groups
	routerGroups, err := e.ReadRouterGroups()
	if err != nil {
		return err
	}
	// check for uniqueness of router group name
	for _, rg := range routerGroups {
		if rg.Guid != routerGroup.Guid && rg.Name == routerGroup.Name {
			msg := fmt.Sprintf("The RouterGroup with name: %s already exists", routerGroup.Name)
			return DBError{Type: UniqueField, Message: msg}
		}
	}

	key := generateRouterGroupKey(routerGroup)
	rg, err := e.storeAdapter.Get(key)
	if err == nil {
		current := models.RouterGroup{}
		json.Unmarshal([]byte(rg.Value), &current)
		if routerGroup.Name != current.Name {
			return DBError{Type: NonUpdatableField, Message: "The RouterGroup Name cannot be updated"}
		}
	}
	json, _ := json.Marshal(routerGroup)
	node := storeadapter.StoreNode{
		Key:   key,
		Value: json,
	}
	return e.storeAdapter.SetMulti([]storeadapter.StoreNode{node})
}

func (e *etcd) ReadRouterGroups() (models.RouterGroups, error) {
	routerGroups, err := e.storeAdapter.ListRecursively(ROUTER_GROUP_BASE_KEY)
	if err != nil {
		return models.RouterGroups{}, nil
	}

	results := []models.RouterGroup{}
	for _, node := range routerGroups.ChildNodes {
		routerGroup := models.RouterGroup{}
		json.Unmarshal([]byte(node.Value), &routerGroup)
		results = append(results, routerGroup)
	}
	return results, nil
}

func generateHttpRouteKey(route models.Route) string {
	return fmt.Sprintf("%s/%s,%s:%d", HTTP_ROUTE_BASE_KEY, url.QueryEscape(route.Route), route.IP, route.Port)
}

func generateRouterGroupKey(routerGroup models.RouterGroup) string {
	return fmt.Sprintf("%s/%s", ROUTER_GROUP_BASE_KEY, routerGroup.Guid)
}

func (e *etcd) ReadTcpRouteMappings() ([]models.TcpRouteMapping, error) {
	tcpMappings, err := e.storeAdapter.ListRecursively(TCP_MAPPING_BASE_KEY)
	if err != nil {
		return []models.TcpRouteMapping{}, nil
	}

	listMappings := []models.TcpRouteMapping{}
	for _, routerGroupNode := range tcpMappings.ChildNodes {
		for _, externalPortNode := range routerGroupNode.ChildNodes {
			for _, mappingNode := range externalPortNode.ChildNodes {
				tcpMapping := models.TcpRouteMapping{}
				json.Unmarshal([]byte(mappingNode.Value), &tcpMapping)
				listMappings = append(listMappings, tcpMapping)
			}
		}
	}
	return listMappings, nil
}

func (e *etcd) SaveTcpRouteMapping(tcpMapping models.TcpRouteMapping) error {
	key := generateTcpRouteMappingKey(tcpMapping)
	tcpMappingJson, _ := json.Marshal(tcpMapping)
	node := storeadapter.StoreNode{
		Key:   key,
		Value: tcpMappingJson,
	}
	return e.storeAdapter.SetMulti([]storeadapter.StoreNode{node})
}

func (e *etcd) DeleteTcpRouteMapping(tcpMapping models.TcpRouteMapping) error {
	key := generateTcpRouteMappingKey(tcpMapping)
	err := e.storeAdapter.Delete(key)
	if err != nil && err.Error() == "the requested key could not be found" {
		err = DBError{Type: KeyNotFound, Message: "The specified route (" + tcpMapping.String() + ") could not be found."}
	}
	return err
}

func generateTcpRouteMappingKey(tcpMapping models.TcpRouteMapping) string {
	// Generating keys following this pattern
	// /v1/tcp_routes/router_groups/{router_guid}/{port}/{host-ip}:{host-port}
	return fmt.Sprintf("%s/%s/%d/%s:%d", TCP_MAPPING_BASE_KEY,
		tcpMapping.TcpRoute.RouterGroupGuid, tcpMapping.TcpRoute.ExternalPort, tcpMapping.HostIP, tcpMapping.HostPort)
}
