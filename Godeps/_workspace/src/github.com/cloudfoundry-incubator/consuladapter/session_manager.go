package consuladapter

import (
	"errors"

	"github.com/hashicorp/consul/api"
)

var ErrPresenceNotSet error = errors.New("Presence not set")

//go:generate counterfeiter -o fakes/fake_session_manager.go . SessionManager
type SessionManager interface {
	NodeName() (string, error)
	Node(node string, q *api.QueryOptions) ([]*api.SessionEntry, *api.QueryMeta, error)
	Create(se *api.SessionEntry, q *api.WriteOptions) (string, *api.WriteMeta, error)
	CreateNoChecks(se *api.SessionEntry, q *api.WriteOptions) (string, *api.WriteMeta, error)
	Destroy(id string, q *api.WriteOptions) (*api.WriteMeta, error)
	Renew(id string, q *api.WriteOptions) (*api.SessionEntry, *api.WriteMeta, error)
	RenewPeriodic(initialTTL string, id string, q *api.WriteOptions, doneCh chan struct{}) error

	NewLock(sessionID, key string, value []byte) (Lock, error)
}

//go:generate counterfeiter -o fakes/fake_lock.go . Lock
type Lock interface {
	Lock(stopCh <-chan struct{}) (lostLock <-chan struct{}, err error)
}

type sessionMgr struct {
	client  *api.Client
	session *api.Session
}

func NewSessionManager(client *api.Client) *sessionMgr {
	return &sessionMgr{
		client:  client,
		session: client.Session(),
	}
}

func (sm *sessionMgr) NodeName() (string, error) {
	return sm.client.Agent().NodeName()
}

func (sm *sessionMgr) Node(node string, q *api.QueryOptions) ([]*api.SessionEntry, *api.QueryMeta, error) {
	return sm.session.Node(node, q)
}

func (sm *sessionMgr) Create(se *api.SessionEntry, q *api.WriteOptions) (string, *api.WriteMeta, error) {
	return sm.session.Create(se, q)
}

func (sm *sessionMgr) CreateNoChecks(se *api.SessionEntry, q *api.WriteOptions) (string, *api.WriteMeta, error) {
	return sm.session.CreateNoChecks(se, q)
}

func (sm *sessionMgr) Destroy(id string, q *api.WriteOptions) (*api.WriteMeta, error) {
	return sm.session.Destroy(id, q)
}

func (sm *sessionMgr) Renew(id string, q *api.WriteOptions) (*api.SessionEntry, *api.WriteMeta, error) {
	return sm.session.Renew(id, q)
}

func (sm *sessionMgr) RenewPeriodic(initialTTL string, id string, q *api.WriteOptions, doneCh chan struct{}) error {
	return sm.session.RenewPeriodic(initialTTL, id, q, doneCh)
}

func (sm *sessionMgr) NewLock(sessionID, key string, value []byte) (Lock, error) {
	lockOptions := api.LockOptions{
		Key:     key,
		Value:   value,
		Session: sessionID,
	}

	return sm.client.LockOpts(&lockOptions)
}
