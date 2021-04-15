package config

import (
	"fmt"
	"strings"
)

type (
	SecretPathFormatter func(secret string) string

	VaultReader struct {
		abstractReader
		storage   *StorageVault
		formatter SecretPathFormatter
		tag       string
	}
)

// reads vault variables to the provided configuration structure
func (r VaultReader) Read(cfg interface{}) error {
	metaInfo, err := readStructMetadata(cfg)
	if err != nil {
		return err
	}

	keyMap := r.storage.InitMemorisedKvMap()

	for _, meta := range metaInfo {
		var val interface{}
		tag, _ := meta.Tag.Lookup(r.tag)
		if tag == "" {
			continue
		}
		vaultTags := strings.Split(tag, ":")
		if len(vaultTags) != 2 {
			return fmt.Errorf("VAULT: %s Tag is invalid", tag)
		}
		key := vaultTags[0]
		if r.formatter != nil {
			key = r.formatter(key)
		}
		if val, err = keyMap(key, vaultTags[1]); err != nil {
			if r.Quiet {
				continue
			}
			return err
		}

		if err = parseValue(meta.FieldValue, val.(string), meta.Separator, meta.Layout); err != nil {
			return err
		}
	}

	return nil
}
