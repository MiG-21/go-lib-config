package config

import (
	"errors"
	"fmt"

	"github.com/hashicorp/vault/api"
)

var (
	errEnvVaultEmptyAddress = errors.New("empty address for vault api")
	errAuthEndpoint         = errors.New("empty auth endpoint")
)

type (
	VaultProviderType string

	VaultAuthenticate interface {
		Authenticate() error
		GetClient() *api.Client
		Stop()
	}

	StorageVault struct {
		VaultAuthenticate
		vaultDataKey string
	}
)

func (st *StorageVault) Read(vaultPath string) (map[string]interface{}, error) {
	if err := st.Authenticate(); err != nil {
		return nil, err
	}

	vaultSecret, err := st.GetClient().Logical().Read(vaultPath)
	if err != nil {
		return nil, err
	}

	if vaultSecret == nil {
		return nil, fmt.Errorf("nil secret on %s", vaultPath)
	}

	if vaultSecret.Data == nil {
		return nil, fmt.Errorf("nil secret.Data on %s", vaultPath)
	}

	return vaultSecret.Data, nil
}

func (st *StorageVault) Write(vaultPath string, data map[string]interface{}) (map[string]interface{}, error) {
	if err := st.Authenticate(); err != nil {
		return nil, err
	}

	vaultSecret, err := st.GetClient().Logical().Write(vaultPath, data)
	if err != nil {
		return nil, err
	}

	if vaultSecret == nil {
		return nil, fmt.Errorf("nil secret on %s", vaultPath)
	}

	if vaultSecret.Data == nil {
		return nil, fmt.Errorf("nil secret.Data on %s", vaultPath)
	}

	return vaultSecret.Data, nil
}

// InitMemorisedKvMap avoid too many allocations by memorizing the "path|key" pair for an event
// @see https://gobyexample.com/closures
func (st *StorageVault) InitMemorisedKvMap() func(path string, key string) (interface{}, error) {
	m := make(map[string]map[string]interface{})
	return func(path string, key string) (interface{}, error) {
		if _, ok := m[path]; !ok {
			if data, err := st.Read(path); err != nil {
				return nil, err
			} else {
				// retrieve data
				var secret interface{}
				if secret, ok = data[st.vaultDataKey]; !ok {
					return nil, fmt.Errorf("failed to get data on %s for %s", path, st.vaultDataKey)
				}
				// cast data
				var secretData map[string]interface{}
				if secretData, ok = secret.(map[string]interface{}); !ok {
					return nil, fmt.Errorf("failed to cast to key-value pairs on %s", path)
				}
				// store data
				m[path] = secretData
			}
		}
		// search in memorized data
		if k, ok := m[path][key]; !ok {
			return nil, fmt.Errorf("nil value on %s:%s", path, key)
		} else {
			return k, nil
		}
	}
}
