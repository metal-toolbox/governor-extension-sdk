package erdscli

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/metal-toolbox/governor-extension-sdk/pkg/erdvalidator"
	"go.uber.org/zap"
)

func validate() error {
	if erdpath == "" {
		return fmt.Errorf("%w: erds-path", ErrValidatorMissingArgs)
	}

	logger.Info("validating ERDs")

	// list files
	files, err := os.ReadDir(erdpath)
	if err != nil {
		return err
	}

	// validate each file
	errchan := make(chan error, len(files))
	wg := &sync.WaitGroup{}

	validateFileAsync := func(path string) {
		defer wg.Done()

		if err := validateFile(path); err != nil {
			errchan <- err
		}
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		path := filepath.Join(erdpath, file.Name())

		wg.Add(1)

		go validateFileAsync(path)
	}

	wg.Wait()
	close(errchan)

	hasErrors := false

	for err := range errchan {
		logger.Error("failed to validate ERD", zap.Error(err))

		hasErrors = true
	}

	if hasErrors {
		os.Exit(1)
	}

	logger.Info("ERDs are valid")

	return nil
}

func validateFile(path string) error {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	ext := filepath.Ext(path)

	var content erdvalidator.ERDContent

	switch ext {
	case ".json":
		content = (*erdvalidator.ERDContentJSON)(&bytes)
	case ".yaml", ".yml":
		content = (*erdvalidator.ERDContentYAML)(&bytes)
	default:
		return fmt.Errorf("%w: %s is not a supported file", ErrFailedToReadFiles, ext)
	}

	v, err := erdvalidator.NewValidator(erdvalidator.WithERDContent(content))
	if err != nil {
		return err
	}

	return v.Validate()
}
