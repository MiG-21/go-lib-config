package main

import (
	"log"
	"os"
	"strings"
	"time"

	libConfig "github.com/MiG-21/go-lib-config"
	"github.com/go-playground/validator/v10"
)

// Validator wrapper example
type Validator struct {
	validator *validator.Validate
}

func (cv *Validator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

func main() {
	defer os.Clearenv()
	_ = os.Setenv("LOG_LEVEL", "INFO")
	libConfig.Verbose = true
	vaultAddress := os.Getenv("VAULT_URL")
	authEndpoint := os.Getenv("VAULT_K8S_MOUNT")
	authRole := os.Getenv("VAULT_K8S_ROLE")
	authTokenPath := os.Getenv("VAULT_AUTH_K8S_TOKEN_PATH")
	env := os.Getenv("ENV")
	stack := os.Getenv("STACK")
	serviceName := os.Getenv("SERVICE")

	formatter := func(env, stack, service string) libConfig.SecretPathFormatter {
		return func(secret string) string {
			parts := []string{"secret", "data"}
			secret = strings.ReplaceAll(secret, "{{.Env}}", env)
			secret = strings.ReplaceAll(secret, "{{.Stack}}", stack)
			secret = strings.ReplaceAll(secret, "{{.Service}}", service)
			parts = append(parts, strings.Trim(secret, "/"))
			return strings.Join(parts, "/")
		}
	}
	cfg := &struct {
		HeadersTimeout string `vault:"{{.Env}}/{{.Stack}}/{{.Service}}/server/connection/headersTimeout:value"`
		LogLevel       string `vault:"{{.Env}}/{{.Stack}}/{{.Service}}/logger/common:level" env:"LOG_LEVEL"`
		SampleRate     int    `vault:"{{.Env}}/{{.Stack}}/{{.Service}}/logger/common:samplerate" data-default:"50"`
		Threshold      int    `vault:"{{.Env}}/{{.Stack}}/{{.Service}}/logger/common:threshold"`
		File           string `vault:"{{.Env}}/{{.Stack}}/{{.Service}}/logger/file:name" validate:"required"`
	}{}
	service := libConfig.NewConfigService(15 * time.Second)
	// assign validator
	service.Validator = &Validator{validator: validator.New()}
	defer func() {
		_ = service.Stop()
	}()
	auth := libConfig.NewVaultK8sAuth(vaultAddress, authEndpoint, authTokenPath, authRole)
	vaultConfig := libConfig.NewApiConfig(vaultAddress, false)
	vault, _ := libConfig.NewStorageVault(auth, vaultConfig, "data")
	vaultReader := libConfig.NewVaultReaderWithFormatter(vault, formatter(env, stack, serviceName))
	envReader := libConfig.NewEnvReader()
	// loop has been started only if config is valid
	_, err := service.Start(cfg, func(valid bool, err error) {
		log.Println("Config has been refreshed")
		if err != nil {
			log.Println(err)
		}
		log.Println(cfg)
	}, vaultReader, envReader)
	if err != nil {
		log.Println(err)
	}
	log.Println(cfg)
}
