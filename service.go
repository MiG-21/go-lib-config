package config

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
)

var (
	LibLogger = defaultLogger
	Verbose   = false
)

type (
	// Validator is the interface that wraps the Validate function.
	Validator interface {
		Validate(i interface{}) error
	}
	// LoadCallback function to handle config refresh error result
	LoadCallback func(valid bool, err error)
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
func (s *Service) Start(cfg interface{}, cb LoadCallback, readers ...Reader) (bool, error) {
	valid, err := s.ReadAndValidate(cfg, readers...)
	// since it is possible to use more than one reader, then there may be a case that
	// having read errors we will receive a valid configuration
	// so run refresh look if acquired config is valid
	if valid {
		s.loop(cfg, cb, readers...)
	}

	return valid, err
}

// ReadAndValidate config
func (s *Service) ReadAndValidate(cfg interface{}, readers ...Reader) (bool, error) {
	var err error
	var errors *multierror.Error
	var metaInfo []*StructMeta

	if len(readers) == 0 {
		return false, fmt.Errorf("no config readers found")
	} else {
		metaInfo, err = ReadStructMetadata(cfg)
		if err != nil {
			return false, err
		}

		for _, reader := range readers {
			if err = reader.Read(metaInfo); err != nil {
				errors = multierror.Append(errors, err)
			}
		}
	}

	if err = setDefaults(metaInfo); err != nil {
		errors = multierror.Append(errors, err)
	}

	valid := true
	if s.Validator != nil {
		if err = s.Validator.Validate(cfg); err != nil {
			errors = multierror.Append(errors, err)
			valid = false
		}
	}

	if errors != nil {
		errors.ErrorFormat = errorFormatter
		err = errors.ErrorOrNil()
	}

	return valid, err
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
					valid, err := s.ReadAndValidate(cfg, readers...)
					if cb != nil {
						cb(valid, err)
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

func defaultLogger(i ...interface{}) {
	if Verbose {
		log.Println(i...)
	}
}

func errorFormatter(es []error) string {
	points := make([]string, len(es))
	for i, err := range es {
		points[i] = err.Error()
	}

	return fmt.Sprintf(
		"%d errors occurred: %s",
		len(es), strings.Join(points, ", "))
}
