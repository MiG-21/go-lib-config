package config

import (
	"fmt"
	"time"

	"github.com/hashicorp/vault/api"
)

type VaultTokenAuth struct {
	token      string
	quit       chan bool
	increment  int
	refreshing bool
	Client     *api.Client
	Secret     *api.Secret
}

func (a *VaultTokenAuth) Authenticate() error {
	if a.Secret == nil && !a.isExpired() {
		if s, err := a.getTokenEntity(); err != nil {
			return err
		} else {
			a.Secret = s
			if token, err := a.Secret.TokenID(); err != nil {
				return err
			} else {
				a.Client.SetToken(token)
				return a.renewToken()
			}
		}
	}
	return nil
}

func (a *VaultTokenAuth) GetClient() *api.Client {
	return a.Client
}

func (a *VaultTokenAuth) getTokenEntity() (*api.Secret, error) {
	a.Client.SetToken(a.token)
	return a.Client.Auth().Token().LookupSelf()
}

func (a *VaultTokenAuth) isExpired() bool {
	if a.Secret != nil {
		renewable, err := a.Secret.TokenIsRenewable()
		if err != nil {
			return false
		}
		if renewable {
			then, err := time.Parse(time.RFC3339Nano, a.Secret.Data["expire_time"].(string))
			if err != nil {
				return false
			}
			return time.Since(then) > 0
		}
	}
	return false
}

func (a *VaultTokenAuth) renewToken() error {
	if a.refreshing {
		return nil
	}

	if a.Secret != nil {
		ttl, err := a.Secret.TokenTTL()
		if err != nil {
			return err
		}
		if ttl == 0 {
			return fmt.Errorf("invalid token TTL")
		}
		nextRead := time.After(ttl / 10)
		go func() {
			a.refreshing = true
			for {
				select {
				case <-a.quit:
					a.refreshing = false
					return
				case <-nextRead:
					self, err := a.Client.Auth().Token().RenewSelf(a.increment)
					if err != nil {
						a.onError(err)
						return
					}
					a.Secret = self
					a.increment++
					ttl, err := a.Secret.TokenTTL()
					if err != nil {
						a.onError(err)
						return
					}
					if ttl == 0 {
						a.onError(fmt.Errorf("invalid token TTL"))
						return
					}
					LibLogger("Vault token has been refreshed")
					nextRead = time.After(ttl / 10)
				}
			}
		}()
	}
	return nil
}

func (a *VaultTokenAuth) Stop() {
	if a.refreshing && a.quit != nil {
		a.quit <- true
		close(a.quit)
	}
}

func (a *VaultTokenAuth) onError(err error) {
	LibLogger(err.Error())
	close(a.quit)
	a.refreshing = false
}
