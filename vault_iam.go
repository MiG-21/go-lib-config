package config

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/hashicorp/vault/api"
)

const (
	VaultAuthHeaderName = "X-Vault-AWS-IAM-Server-ID"
)

type (
	VaultIAMAuth struct {
		VaultTokenAuth `json:"-"`

		httpClient      *http.Client
		vaultAddress    string
		vaultAuthMount  string
		vaultAuthRole   string
		vaultAuthHeader string
	}
)

func (a *VaultIAMAuth) getAuthUrl() (string, error) {
	if a.vaultAddress == "" {
		return "", errEnvVaultEmptyAddress
	}

	if a.vaultAuthMount == "" {
		return "", errAuthMount
	}

	return strings.TrimRight(a.vaultAddress, "/") +
		path.Join("/v1/auth", strings.Trim(a.vaultAuthMount, "/"), "login"), nil
}

func (a *VaultIAMAuth) sendAuthRequest() (*api.Secret, error) {
	URL, err := a.getAuthUrl()
	if err != nil {
		return nil, err
	}

	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	stsSvc := sts.New(sess)
	req, _ := stsSvc.GetCallerIdentityRequest(&sts.GetCallerIdentityInput{})

	if a.vaultAuthHeader != "" {
		// if supplied, and then sign the request including that header
		req.HTTPRequest.Header.Add(VaultAuthHeaderName, a.vaultAuthHeader)
	}
	if err = req.Sign(); err != nil {
		return nil, err
	}

	headers, err := json.Marshal(req.HTTPRequest.Header)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(req.HTTPRequest.Body)
	if err != nil {
		return nil, err
	}

	d := make(map[string]interface{})
	d["iam_http_request_method"] = req.HTTPRequest.Method
	d["iam_request_url"] = base64.StdEncoding.EncodeToString([]byte(req.HTTPRequest.URL.String()))
	d["iam_request_headers"] = base64.StdEncoding.EncodeToString(headers)
	d["iam_request_body"] = base64.StdEncoding.EncodeToString(body)
	d["role"] = a.vaultAuthRole

	resp, err := a.Client.Logical().Write(URL, d)
	if err != nil {
		return nil, err
	}

	if token, err := resp.TokenID(); err != nil {
		return nil, err
	} else {
		a.GetClient().SetToken(token)
	}

	return a.getTokenEntity()
}

func (a *VaultIAMAuth) Authenticate() error {
	if a.Secret == nil || a.isExpired() {
		auth, err := a.sendAuthRequest()
		if err != nil {
			return err
		}
		a.Secret = auth
		if token, err := a.Secret.TokenID(); err != nil {
			return err
		} else {
			a.Client.SetToken(token)
			return a.renewToken()
		}
	}
	return nil
}
