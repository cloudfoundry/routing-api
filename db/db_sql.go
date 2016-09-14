package db

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/coreos/etcd/Godeps/_workspace/src/golang.org/x/net/context"

	"code.cloudfoundry.org/eventhub"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/routing-api/config"
	"code.cloudfoundry.org/routing-api/models"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

type SqlDB struct {
	Client       Client
	tcpEventHub  eventhub.Hub
	httpEventHub eventhub.Hub
}

const DeleteError = "Delete Fails: Route does not exist"

var _ DB = &SqlDB{}

func NewSqlDB(cfg *config.SqlDB) (DB, error) {
	if cfg == nil {
		return nil, errors.New("SQL configuration cannot be nil")
	}
	connectionString := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Schema)

	db, err := gorm.Open(cfg.Type, connectionString)
	if err != nil {
		return nil, err
	}

	db.AutoMigrate(&models.RouterGroupDB{}, &models.TcpRouteMapping{}, &models.Route{})

	tcpEventHub := eventhub.NewNonBlocking(1024)
	httpEventHub := eventhub.NewNonBlocking(1024)

	return &SqlDB{Client: db, tcpEventHub: tcpEventHub, httpEventHub: httpEventHub}, nil
}

func (s *SqlDB) CleanupRoutes(logger lager.Logger, pruningInterval time.Duration, signals <-chan os.Signal) {
	pruningTicker := time.NewTicker(pruningInterval)
	for {
		select {
		case <-pruningTicker.C:
			db := s.Client.Where("expires_at < ?", time.Now()).Delete(models.TcpRouteMapping{})
			if db.Error != nil {
				logger.Error("failed-to-prune-routes", db.Error)
			} else {
				logger.Info("successfully-finished-pruning", lager.Data{"rowsAffected": db.RowsAffected})
			}
		case <-signals:
			return
		}
	}
}

func (s *SqlDB) ReadRouterGroups() (models.RouterGroups, error) {
	routerGroupsDB := models.RouterGroupsDB{}
	routerGroups := models.RouterGroups{}
	err := s.Client.Find(&routerGroupsDB).Error
	if err == nil {
		routerGroups = routerGroupsDB.ToRouterGroups()
	}

	return routerGroups, err
}

func (s *SqlDB) ReadRouterGroup(guid string) (models.RouterGroup, error) {
	routerGroupDB := models.RouterGroupDB{}
	routerGroup := models.RouterGroup{}
	err := s.Client.Where("guid = ?", guid).First(&routerGroupDB).Error
	if err == nil {
		routerGroup = routerGroupDB.ToRouterGroup()
	}

	if recordNotFound(err) {
		err = nil
	}
	return routerGroup, err
}

func (s *SqlDB) SaveRouterGroup(routerGroup models.RouterGroup) error {
	existingRouterGroup, err := s.ReadRouterGroup(routerGroup.Guid)
	if err != nil {
		return err
	}

	routerGroupDB := models.NewRouterGroupDB(routerGroup)
	if existingRouterGroup.Guid == routerGroup.Guid {
		updateRouterGroup(&existingRouterGroup, &routerGroup)
		routerGroupDB = models.NewRouterGroupDB(existingRouterGroup)
		err = s.Client.Save(&routerGroupDB).Error
	} else {
		err = s.Client.Create(&routerGroupDB).Error
	}

	return err
}

func updateRouterGroup(existingRouterGroup, currentRouterGroup *models.RouterGroup) {
	if currentRouterGroup.Type != "" {
		existingRouterGroup.Type = currentRouterGroup.Type
	}
	if currentRouterGroup.Name != "" {
		existingRouterGroup.Name = currentRouterGroup.Name
	}
	if currentRouterGroup.ReservablePorts != "" {
		existingRouterGroup.ReservablePorts = currentRouterGroup.ReservablePorts
	}
}

func updateTcpRouteMapping(existingTcpRouteMapping models.TcpRouteMapping, currentTcpRouteMapping models.TcpRouteMapping) models.TcpRouteMapping {
	existingTcpRouteMapping.ModificationTag.Increment()
	if currentTcpRouteMapping.TTL != nil {
		existingTcpRouteMapping.TTL = currentTcpRouteMapping.TTL
	}

	existingTcpRouteMapping.ExpiresAt = time.Now().
		Add(time.Duration(*existingTcpRouteMapping.TTL) * time.Second)

	return existingTcpRouteMapping
}

func updateRoute(existingRoute, currentRoute models.Route) models.Route {
	existingRoute.ModificationTag.Increment()
	if currentRoute.TTL != nil {
		existingRoute.TTL = currentRoute.TTL
	}

	if currentRoute.LogGuid != "" {
		existingRoute.LogGuid = currentRoute.LogGuid
	}

	existingRoute.ExpiresAt = time.Now().
		Add(time.Duration(*existingRoute.TTL) * time.Second)

	return existingRoute
}

func notImplementedError() error {
	pc, _, _, _ := runtime.Caller(1)
	fnName := runtime.FuncForPC(pc).Name()
	return errors.New(fmt.Sprintf("function not implemented: %s", fnName))
}

func (s *SqlDB) ReadRoutes() ([]models.Route, error) {
	var routes []models.Route
	now := time.Now()
	err := s.Client.Where("expires_at > ?", now).Find(&routes).Error
	if err != nil {
		return nil, err
	}
	return routes, err
}

func (s *SqlDB) readRoute(route models.Route) (models.Route, error) {
	var routes []models.Route
	err := s.Client.Where("route = ? and ip = ? and port = ? and route_service_url = ?",
		route.Route, route.IP, route.Port, route.RouteServiceUrl).Find(&routes).Error

	if err != nil {
		return route, err
	}
	count := len(routes)
	if count > 1 || count < 0 {
		return route, errors.New("Have duplicate routes")
	}
	if count == 1 {
		return routes[0], nil
	}
	return models.Route{}, nil
}

func (s *SqlDB) SaveRoute(route models.Route) error {
	existingRoute, err := s.readRoute(route)
	if err != nil {
		return nil
	}

	if existingRoute != (models.Route{}) {
		newRoute := updateRoute(existingRoute, route)
		err = s.Client.Save(&newRoute).Error
		if err != nil {
			return err
		}
		return s.emitEvent(UpdateEvent, newRoute)
	}

	newRoute, err := models.NewRouteWithModel(route)
	if err != nil {
		return err
	}

	tag, err := models.NewModificationTag()
	if err != nil {
		return err
	}
	newRoute.ModificationTag = tag

	err = s.Client.Create(&newRoute).Error
	if err != nil {
		return err
	}
	return s.emitEvent(CreateEvent, newRoute)
}

func (s *SqlDB) DeleteRoute(route models.Route) error {
	route, err := s.readRoute(route)
	if err != nil {
		return err
	}
	if route == (models.Route{}) {
		return errors.New(DeleteError)
	}

	err = s.Client.Delete(&route).Error
	if err != nil {
		return err
	}
	return s.emitEvent(DeleteEvent, route)
}

func (s *SqlDB) ReadTcpRouteMappings() ([]models.TcpRouteMapping, error) {
	var tcpRoutes []models.TcpRouteMapping
	now := time.Now()
	err := s.Client.Where("expires_at > ?", now).Find(&tcpRoutes).Error
	if err != nil {
		return nil, err
	}
	return tcpRoutes, nil
}

func (s *SqlDB) readTcpRouteMapping(tcpMapping models.TcpRouteMapping) (models.TcpRouteMapping, error) {
	var routes []models.TcpRouteMapping
	var tcpRoute models.TcpRouteMapping
	err := s.Client.Where("host_ip = ? and host_port = ? and external_port = ?",
		tcpMapping.HostIP, tcpMapping.HostPort, tcpMapping.ExternalPort).Find(&routes).Error

	if err != nil {
		return tcpRoute, err
	}
	count := len(routes)
	if count > 1 || count < 0 {
		return tcpRoute, errors.New("Have duplicate tcp route mappings")
	}
	if count == 1 {
		tcpRoute = routes[0]
	}

	return tcpRoute, err
}

func (s *SqlDB) emitEvent(eventType EventType, obj interface{}) error {
	event, err := NewEventFromInterface(eventType, obj)
	if err != nil {
		return err
	}

	switch obj.(type) {
	case models.Route:
		s.httpEventHub.Emit(event)
	case models.TcpRouteMapping:
		s.tcpEventHub.Emit(event)
	default:
		return errors.New("Unknown event type")
	}
	return nil
}

func (s *SqlDB) SaveTcpRouteMapping(tcpRouteMapping models.TcpRouteMapping) error {
	existingTcpRouteMapping, err := s.readTcpRouteMapping(tcpRouteMapping)
	if err != nil {
		return err
	}

	if existingTcpRouteMapping != (models.TcpRouteMapping{}) {
		newTcpRouteMapping := updateTcpRouteMapping(existingTcpRouteMapping, tcpRouteMapping)
		err = s.Client.Save(&newTcpRouteMapping).Error
		if err != nil {
			return err
		}
		return s.emitEvent(UpdateEvent, newTcpRouteMapping)
	}

	tcpMapping, err := models.NewTcpRouteMappingWithModel(tcpRouteMapping)
	if err != nil {
		return err
	}

	tag, err := models.NewModificationTag()
	if err != nil {
		return err
	}
	tcpMapping.ModificationTag = tag

	err = s.Client.Create(&tcpMapping).Error
	if err != nil {
		return err
	}

	return s.emitEvent(CreateEvent, tcpMapping)
}

func (s *SqlDB) DeleteTcpRouteMapping(tcpMapping models.TcpRouteMapping) error {
	tcpMapping, err := s.readTcpRouteMapping(tcpMapping)
	if err != nil {
		return err
	}
	if tcpMapping == (models.TcpRouteMapping{}) {
		return errors.New(DeleteError)
	}

	err = s.Client.Delete(&tcpMapping).Error
	if err != nil {
		return err
	}
	return s.emitEvent(DeleteEvent, tcpMapping)
}

func (s *SqlDB) Connect() error {
	return notImplementedError()
}

func (s *SqlDB) CancelWatches() {
	// This only errors if the eventhub was closed.
	_ = s.tcpEventHub.Close()
	_ = s.httpEventHub.Close()
}

func (s *SqlDB) WatchRouteChanges(watchType string) (<-chan Event, <-chan error, context.CancelFunc) {
	var (
		sub eventhub.Source
		err error
	)
	events := make(chan Event)
	errors := make(chan error, 1)
	cancelFunc := func() {}

	switch watchType {
	case TCP_WATCH:
		sub, err = s.tcpEventHub.Subscribe()
		if err != nil {
			errors <- err
			close(events)
			close(errors)
			return events, errors, cancelFunc
		}
	case HTTP_WATCH:
		sub, err = s.httpEventHub.Subscribe()
		if err != nil {
			errors <- err
			close(events)
			close(errors)
			return events, errors, cancelFunc
		}
	default:
		err := fmt.Errorf("Invalid watch type: %s", watchType)
		errors <- err
		close(events)
		close(errors)
		return events, errors, cancelFunc
	}

	cancelFunc = func() {
		sub.Close()
	}

	go dispatchWatchEvents(sub, events, errors)

	return events, errors, cancelFunc
}

func dispatchWatchEvents(sub eventhub.Source, events chan<- Event, errors chan<- error) {
	defer close(events)
	defer close(errors)
	for {
		event, err := sub.Next()
		if err != nil {
			if err == eventhub.ErrReadFromClosedSource {
				return
			}
			errors <- err
		}
		watchEvent, ok := event.(Event)
		if !ok {
			errors <- fmt.Errorf("Incoming event is not a db.Event: %#v", event)
		}
		events <- watchEvent
	}
}

func recordNotFound(err error) bool {
	if err == gorm.ErrRecordNotFound {
		return true
	}
	return false
}
