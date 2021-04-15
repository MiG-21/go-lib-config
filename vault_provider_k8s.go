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
)

var (
	errK8sAuthBadResponseStatusCode = errors.New("bad response status code")
	errK8sAuthBadResponseBody       = errors.New("bad response body")
	errK8sAuthEmptyClientToken      = errors.New("empty auth.client_token property in response body")
	errK8sAuthEndpoint              = errors.New("empty auth endpoint")
)

type (
	vaultK8sLoginResponse struct {
		Auth struct {
			ClientToken   string   `json:"client_token"`
			Accessor      string   `json:"accessor"`
			Policies      []string `json:"policies"`
			LeaseDuration int      `json:"lease_duration"`
			Renewable     bool     `json:"renewable"`
			Metadata      struct {
				Role                     string `json:"role"`
				ServiceAccountName       string `json:"service_account_name"`
				ServiceAccountNamespace  string `json:"service_account_namespace"`
				ServiceAccountSecretName string `json:"service_account_secret_name"`
				ServiceAccountUID        string `json:"service_account_uid"`
			} `json:"metadata"`
		} `json:"auth"`
	}

	VaultK8sAuth struct {
		Role     string                `json:"role"`
		JWT      string                `json:"jwt"`
		Response vaultK8sLoginResponse `json:"-"`

		httpClient        *http.Client
		vaultAddress      string
		vaultAuthEndpoint string
		tokenPath         string
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

	if a.vaultAuthEndpoint == "" {
		return "", errK8sAuthEndpoint
	}

	return strings.TrimRight(a.vaultAddress, "/") +
		path.Join("/v1/auth", strings.Trim(a.vaultAuthEndpoint, "/"), "login"), nil
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

func (a *VaultK8sAuth) parseResponseToken(res *http.Response) (string, error) {
	defer func() {
		_ = res.Body.Close()
	}()

	buff := bytes.NewBuffer([]byte{})

	if _, err := buff.ReadFrom(res.Body); err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w; statusCode %d for k8s a requestToken", errK8sAuthBadResponseStatusCode, res.StatusCode)
	}

	result := vaultK8sLoginResponse{}

	if err := json.Unmarshal(buff.Bytes(), &result); err != nil {
		return "", fmt.Errorf("%w: %s", errK8sAuthBadResponseBody, err)
	} else {
		a.Response = result
	}

	if a.Response.Auth.ClientToken == "" {
		return "", errK8sAuthEmptyClientToken
	}

	return a.Response.Auth.ClientToken, nil
}

func (a *VaultK8sAuth) GetToken() (string, error) {
	// @TODO this part should be improved
	res, err := a.sendAuthRequest()
	if err != nil {
		return "", err
	}

	return a.parseResponseToken(res)
}
