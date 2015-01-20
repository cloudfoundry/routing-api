package db

import (
	"encoding/json"
	"fmt"

	etcdclient "github.com/coreos/go-etcd/etcd"
)

//go:generate counterfeiter -o fakes/fake_db.go . DB
type DB interface {
	SaveRoute(route Route) error
}

type Route struct {
	Route   string `json:"route"`
	Port    int    `json:"port"`
	IP      string `json:"ip"`
	TTL     int    `json:"ttl"`
	LogGuid string `json:"log_guid"`
}

type etcd struct {
	client *etcdclient.Client
}

func NewETCD(nodeURLs []string) etcd {
	return etcd{
		client: etcdclient.NewClient(nodeURLs),
	}
}

func (e etcd) SaveRoute(route Route) error {
	key := fmt.Sprintf("/routes/%s,%s:%d", route.Route, route.IP, route.Port)
	routeJSON, _ := json.Marshal(route)
	_, err := e.client.Set(key, string(routeJSON), uint64(route.TTL))

	return err
}
