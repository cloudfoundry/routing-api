package consuladapter

import "fmt"

func (s *Session) GetAcquiredValue(key string) ([]byte, error) {
	kvPair, _, err := s.kv.Get(key, nil)
	if err != nil {
		return nil, err
	}

	if kvPair == nil || kvPair.Session == "" {
		return nil, NewKeyNotFoundError(key)
	}

	return kvPair.Value, nil
}

func (s *Session) ListAcquiredValues(prefix string) (map[string][]byte, error) {
	kvPairs, _, err := s.kv.List(prefix, nil)
	if err != nil {
		return nil, err
	}

	if kvPairs == nil {
		return nil, NewPrefixNotFoundError(prefix)
	}

	children := map[string][]byte{}
	for _, kvPair := range kvPairs {
		if kvPair.Session != "" {
			children[kvPair.Key] = kvPair.Value
		}
	}

	return children, nil
}

func NewKeyNotFoundError(key string) error {
	return KeyNotFoundError(key)
}

type KeyNotFoundError string

func (e KeyNotFoundError) Error() string {
	return fmt.Sprintf("key not found: '%s'", string(e))
}

func NewPrefixNotFoundError(prefix string) error {
	return PrefixNotFoundError(prefix)
}

type PrefixNotFoundError string

func (e PrefixNotFoundError) Error() string {
	return fmt.Sprintf("prefix not found: '%s'", string(e))
}
