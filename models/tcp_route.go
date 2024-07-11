package models

import (
	"fmt"
	"time"

	uuid "github.com/nu7hatch/gouuid"
)

type TcpRouteMapping struct {
	Model
	ExpiresAt time.Time `json:"-"`
	TcpMappingEntity
}

type TcpMappingEntity struct {
	RouterGroupGuid string  `gorm:"not null; unique_index:idx_tcp_route" json:"router_group_guid"`
	HostPort        uint16  `gorm:"not null; unique_index:idx_tcp_route; type:int" json:"backend_port"`
	HostTLSPort     *uint16 `gorm:"default:null; unique_index:idx_tcp_route; type:int" json:"backend_tls_port"`
	HostIP          string  `gorm:"not null; unique_index:idx_tcp_route" json:"backend_ip"`
	SniHostname     *string `gorm:"default:null; unique_index:idx_tcp_route" json:"backend_sni_hostname,omitempty"`
	// We don't add uniqueness on InstanceId so that if a route is attempted to be created with the same detals but
	// different InstanceId, we fail uniqueness and prevent stale/duplicate routes. If this fails a route, the
	// TTL on the old record should expire + allow the new route to be created eventually.
	InstanceId       string `gorm:"not null;" json:"instance_id"`
	ExternalPort     uint16 `gorm:"not null; unique_index:idx_tcp_route; type: int" json:"port"`
	ModificationTag  `json:"modification_tag"`
	TTL              *int   `json:"ttl,omitempty"`
	IsolationSegment string `json:"isolation_segment"`
}

func (TcpRouteMapping) TableName() string {
	return "tcp_routes"
}

func NewTcpRouteMappingWithModel(tcpMapping TcpRouteMapping) (TcpRouteMapping, error) {
	guid, err := uuid.NewV4()
	if err != nil {
		return TcpRouteMapping{}, err
	}

	m := Model{Guid: guid.String()}
	return TcpRouteMapping{
		ExpiresAt:        time.Now().Add(time.Duration(*tcpMapping.TTL) * time.Second),
		Model:            m,
		TcpMappingEntity: tcpMapping.TcpMappingEntity,
	}, nil
}

func NewTcpRouteMapping(
	routerGroupGuid string,
	externalPort uint16,
	hostIP string,
	hostPort uint16,
	hostTlsPort uint16,
	instanceId string,
	sniHostname *string,
	ttl int,
	modTag ModificationTag) TcpRouteMapping {
	mapping := TcpRouteMapping{
		TcpMappingEntity: TcpMappingEntity{
			RouterGroupGuid: routerGroupGuid,
			ExternalPort:    externalPort,
			SniHostname:     sniHostname,
			InstanceId:      instanceId,
			HostPort:        hostPort,
			HostTLSPort:     &hostTlsPort,
			HostIP:          hostIP,
			TTL:             &ttl,
			ModificationTag: modTag,
		},
	}
	return mapping
}

func (m TcpRouteMapping) String() string {
	return fmt.Sprintf("%s:%d<->%s:%d", m.RouterGroupGuid, m.ExternalPort, m.HostIP, m.HostPort)
}

func (m TcpRouteMapping) Matches(other TcpRouteMapping) bool {
	return m.RouterGroupGuid == other.RouterGroupGuid &&
		m.ExternalPort == other.ExternalPort &&
		m.HostIP == other.HostIP &&
		m.HostPort == other.HostPort &&
		((m.HostTLSPort == other.HostTLSPort) ||
			m.HostTLSPort != nil && other.HostTLSPort != nil && *m.HostTLSPort == *other.HostTLSPort) &&
		m.InstanceId == other.InstanceId &&
		*m.TTL == *other.TTL &&
		((m.SniHostname == other.SniHostname) ||
			m.SniHostname != nil && other.SniHostname != nil && *m.SniHostname == *other.SniHostname)
}

func (t *TcpRouteMapping) SetDefaults(maxTTL int) {
	// default ttl if not present
	// TTL is a pointer to a uint16 so that we can
	// detect if it's present or not (i.e. nil or 0)
	if t.TTL == nil {
		t.TTL = &maxTTL
	}
}
