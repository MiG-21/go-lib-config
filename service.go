package config

import (
	"fmt"
	"time"
)

type (
	// Validator is the interface that wraps the Validate function.
	Validator interface {
		Validate(i interface{}) error
	}
	// LoadCallback function to handle config refresh error result
	LoadCallback func(err error)
	// Service config options
	Service struct {
		quit chan bool
		// refresh interval
		interval time.Duration
		// state flag
		started bool
		// config validator
		Validator Validator
	}
)

// Start start config service
func (s *Service) Start(cfg interface{}, cb LoadCallback, readers ...Reader) error {
	err := s.ReadAndValidate(cfg, readers...)
	s.loop(cfg, cb, readers...)
	return err
}

// ReadAndValidate config
func (s *Service) ReadAndValidate(cfg interface{}, readers ...Reader) error {
	var err error
	if len(readers) == 0 {
		return fmt.Errorf("no config readers found")
	} else {
		for _, reader := range readers {
			if err = reader.Read(cfg); err != nil {
				return err
			}
		}
	}
	if err = setDefaults(cfg); err != nil {
		return err
	}

	if s.Validator != nil {
		return s.Validator.Validate(cfg)
	}

	return nil
}

// loop run config refresh
func (s *Service) loop(cfg interface{}, cb LoadCallback, readers ...Reader) {
	// start loop if time duration > 0 and not started yet
	if !s.started && s.interval > 0 {
		nextRead := time.After(s.interval)
		go func() {
			for {
				select {
				case <-s.quit:
					s.started = false
					return
				case <-nextRead:
					err := s.ReadAndValidate(cfg, readers...)
					if cb != nil {
						cb(err)
					}
					nextRead = time.After(s.interval)
				}
			}
		}()
		s.started = true
	}
}

// Stop config service
func (s *Service) Stop() error {
	if s.started && s.quit != nil {
		s.quit <- true
		close(s.quit)
	}

	return nil
}
