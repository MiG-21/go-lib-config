package config_test

import (
	"log"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Suite")
}

func timeFunc(s, l string) time.Time {
	tm, err := time.Parse(l, s)
	if err != nil {
		log.Fatal(err)
	}
	return tm
}

func durationFunc(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		log.Fatal(err)
	}
	return d
}

func setEnv(vars map[string]string) {
	for k, v := range vars {
		_ = os.Setenv(k, v)
	}
}
