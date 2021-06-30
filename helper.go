package config

import (
	"net/http"
	"time"

	"github.com/hashicorp/vault/api"
)

func NewVaultTokenAuth(token string) *VaultTokenAuth {
	return &VaultTokenAuth{token: token}
}

func NewVaultK8sAuth(vaultAddress, vaultAuthEndpoint, tokenPath, role string) *VaultK8sAuth {
	return &VaultK8sAuth{
		Role:              role,
		vaultAddress:      vaultAddress,
		vaultAuthEndpoint: vaultAuthEndpoint,
		tokenPath:         tokenPath,
		httpClient: &http.Client{
			Timeout: time.Second * 10,
		},
	}
}

func NewApiConfig(address string, agent bool) *api.Config {
	config := &api.Config{
		HttpClient: &http.Client{
			Timeout: time.Second * 10,
		},
	}
	if agent {
		config.AgentAddress = address
	} else {
		config.Address = address
	}

	return config
}

func NewStorageVault(auth VaultAuthenticate, vaultConfig *api.Config, vaultDataKey string) (*StorageVault, error) {
	vaultClient, err := api.NewClient(vaultConfig)
	if err != nil {
		return nil, err
	}

	return &StorageVault{
		VaultAuthenticate: auth,
		vaultClient:       vaultClient,
		vaultDataKey:      vaultDataKey,
	}, nil
}

func NewEnvReader() EnvReader {
	return EnvReader{
		tag: "env",
	}
}

func NewVaultReader(storage *StorageVault) VaultReader {
	return VaultReader{
		storage: storage,
		tag:     "vault",
	}
}

func NewVaultReaderWithFormatter(storage *StorageVault, formatter SecretPathFormatter) VaultReader {
	reader := NewVaultReader(storage)
	reader.formatter = formatter
	return reader
}

func NewConfigService(interval time.Duration) *Service {
	service := &Service{}
	if interval > 0 {
		service.quit = make(chan bool)
		service.interval = interval
	}
	return service
}
