package config

type VaultTokenAuth struct {
	token string
}

func (a *VaultTokenAuth) GetToken() (string, error) {
	return a.token, nil
}
