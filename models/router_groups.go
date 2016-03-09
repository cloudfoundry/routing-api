package models

type RouterGroupType string

type ReservablePorts string

// func (p ReservablePorts) Validate() error {
// 	return nil
// }

type RouterGroup struct {
	Guid            string          `json:"guid"`
	Name            string          `json:"name"`
	Type            RouterGroupType `json:"type"`
	ReservablePorts ReservablePorts `json:"reservable_ports" yaml:"reservable_ports"`
}

type RouterGroups []RouterGroup
