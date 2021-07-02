package config

import (
	"net/http"
	"time"

	"github.com/hashicorp/vault/api"
)

func NewVaultTokenAuth(token string, vaultConfig *api.Config) (*VaultTokenAuth, error) {
	vaultClient, err := api.NewClient(vaultConfig)
	if err != nil {
		return nil, err
	}
	return &VaultTokenAuth{
		token:  token,
		Client: vaultClient,
		quit:   make(chan bool),
	}, nil
}

func NewVaultK8sAuth(vaultAddress, vaultAuthEndpoint, tokenPath, role string, vaultConfig *api.Config) (*VaultK8sAuth, error) {
	vaultTokenAuth, err := NewVaultTokenAuth("", vaultConfig)
	if err != nil {
		return nil, err
	}
	return &VaultK8sAuth{
		Role:              role,
		vaultAddress:      vaultAddress,
		vaultAuthEndpoint: vaultAuthEndpoint,
		tokenPath:         tokenPath,
		httpClient: &http.Client{
			Timeout: time.Second * 10,
		},
		VaultTokenAuth: *vaultTokenAuth,
	}, nil
}

func NewVaultApiConfig(address string, agent bool) *api.Config {
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

func NewStorageVault(auth VaultAuthenticate, vaultDataKey string) (*StorageVault, error) {
	return &StorageVault{
		VaultAuthenticate: auth,
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
