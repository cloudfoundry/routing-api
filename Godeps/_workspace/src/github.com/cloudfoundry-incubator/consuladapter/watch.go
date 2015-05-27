package consuladapter

import (
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pivotal-golang/lager"
)

const defaultWatchBlockDuration = 10 * time.Second

var emptyBytes = []byte{}

func (s *Session) WatchForDisappearancesUnder(logger lager.Logger, prefix string) <-chan []string {
	logger = logger.Session("watch-for-disappearances-under", lager.Data{"prefix": prefix})

	disappearanceChan := make(chan []string)

	go func() {
		defer func() { close(disappearanceChan) }()

		keys := keySet{}

		queryOpts := &api.QueryOptions{
			WaitIndex: 0,
			WaitTime:  defaultWatchBlockDuration,
		}

		for {
			newPairs, queryMeta, err := s.kv.List(prefix, queryOpts)
			if err != nil {
				logger.Error("list-failed", err)
				select {
				case <-s.doneCh:
					return
				case <-time.After(1 * time.Second):
				}
				queryOpts.WaitIndex = 0
				continue
			}

			select {
			case <-s.doneCh:
				return
			default:
			}

			queryOpts.WaitIndex = queryMeta.LastIndex

			if newPairs == nil {
				// key not found
				_, err = s.kv.Put(&api.KVPair{Key: prefix, Value: emptyBytes}, nil)
				if err != nil {
					logger.Error("put-failed", err)
					continue
				}
			}

			newKeys := newKeySet(newPairs)
			if missing := difference(keys, newKeys); len(missing) > 0 {
				select {
				case disappearanceChan <- missing:
				case <-s.doneCh:
					return
				}
			}

			keys = newKeys
		}
	}()

	return disappearanceChan
}

func newKeySet(keyPairs api.KVPairs) keySet {
	newKeySet := keySet{}
	for _, kvPair := range keyPairs {
		if kvPair.Session != "" {
			newKeySet[kvPair.Key] = struct{}{}
		}
	}
	return newKeySet
}

type keySet map[string]struct{}

func difference(a, b keySet) []string {
	var missing []string
	for key, _ := range a {
		if _, ok := b[key]; !ok {
			missing = append(missing, key)
		}
	}

	return missing
}
