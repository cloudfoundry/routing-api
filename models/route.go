package models

type Route struct {
	Route           string `json:"route"`
	Port            uint16 `json:"port"`
	IP              string `json:"ip"`
	TTL             int    `json:"ttl"`
	LogGuid         string `json:"log_guid"`
	RouteServiceUrl string `json:"route_service_url,omitempty"`
}
