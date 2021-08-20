package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/hashicorp/vault/api"
)

var (
	errK8sAuthBadResponseStatusCode = errors.New("bad response status code")
	errK8sAuthBadResponseBody       = errors.New("bad response body")
	errK8sAuthEmptyClientToken      = errors.New("empty auth.client_token property in response body")
)

type (
	vaultK8sLoginResponse struct {
		Auth api.SecretAuth `json:"auth"`
	}

	VaultK8sAuth struct {
		VaultTokenAuth `json:"-"`

		Role string `json:"role"`
		JWT  string `json:"jwt"`

		httpClient     *http.Client
		vaultAddress   string
		vaultAuthMount string
		tokenPath      string
	}
)

func (a *VaultK8sAuth) readK8sJwtToken() error {
	fp, err := os.OpenFile(a.tokenPath, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer func() {
		_ = fp.Close()
	}()

	info, err := fp.Stat()
	if err != nil {
		return err
	}

	data := make([]byte, info.Size())

	_, err = fp.Read(data)
	if err != nil {
		return err
	}

	a.JWT = string(data)

	return nil
}

func (a *VaultK8sAuth) getAuthUrl() (string, error) {
	if a.vaultAddress == "" {
		return "", errEnvVaultEmptyAddress
	}

	if a.vaultAuthMount == "" {
		return "", errAuthMount
	}

	return strings.TrimRight(a.vaultAddress, "/") +
		path.Join("/v1/auth", strings.Trim(a.vaultAuthMount, "/"), "login"), nil
}

func (a *VaultK8sAuth) sendAuthRequest() (*http.Response, error) {
	err := a.readK8sJwtToken()
	if err != nil {
		return nil, err
	}

	URL, err := a.getAuthUrl()

	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	return a.httpClient.Do(req)

}

func (a *VaultK8sAuth) parseResponseToken(res *http.Response) (*api.Secret, error) {
	defer func() {
		_ = res.Body.Close()
	}()

	buff := bytes.NewBuffer([]byte{})

	if _, err := buff.ReadFrom(res.Body); err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w; statusCode %d for k8s a requestToken", errK8sAuthBadResponseStatusCode, res.StatusCode)
	}

	result := vaultK8sLoginResponse{}

	if err := json.Unmarshal(buff.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("%w: %s", errK8sAuthBadResponseBody, err)
	}

	if result.Auth.ClientToken == "" {
		return nil, errK8sAuthEmptyClientToken
	} else {
		a.GetClient().SetToken(result.Auth.ClientToken)
	}

	return a.getTokenEntity()
}

func (a *VaultK8sAuth) Authenticate() error {
	if a.Secret == nil || a.isExpired() {
		res, err := a.sendAuthRequest()
		if err != nil {
			return err
		}
		if auth, err := a.parseResponseToken(res); err != nil {
			return err
		} else {
			a.Secret = auth
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
