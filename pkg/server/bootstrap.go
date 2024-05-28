package server

import (
	"context"
	"os"
	"path/filepath"

	"github.com/metal-toolbox/governor-api/pkg/api/v1alpha1"
	govclient "github.com/metal-toolbox/governor-api/pkg/client"
	"github.com/metal-toolbox/governor-extension-sdk/pkg/erdvalidator"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Bootstrap is a function that bootstraps the extension
// it is called by the server after it has started
//
//  1. extension check if it is registered
//  2. extension check if it is enabled
//  3. compare local ERDs with ERDs from governor, only create new ERDs if
//     they don't exist in governor. Since ERDs are immutable, the extension will
//     not attempt to update the ERDs if it was changed by the developer.
func (s *Server) Bootstrap(ctx context.Context) error {
	s.status = StatusBootstrapping
	s.logger.Info("bootstrapping extension")

	ctx, span := s.tracer.Start(ctx, "boostrap")
	defer span.End()

	ext, err := s.governorClient.Extension(ctx, s.extensionID, false)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	s.extension = ext

	s.logger.Debug("extension info", zap.Any("extension", ext))

	// list ERDs
	s.logger.Debug("listing extension resources")

	var (
		localERDs  []*v1alpha1.ExtensionResourceDefinitionReq
		govERDsSet map[string]byte
	)

	// a. list local ERDs
	readlocalCtx, span := s.tracer.Start(ctx, "list-local-erds")
	defer span.End()

	localERDs, err = s.readERDsFromLocalDir(readlocalCtx, s.erdDir)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	span.End()

	// b. list ERDs from governor
	listremoteCtx, span := s.tracer.Start(ctx, "list-governor-erds")
	defer span.End()

	govERDsSet, err = listERDsFromGovernor(listremoteCtx, s.governorClient, s.extensionID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	span.End()

	// c. compare local ERDs with ERDs from governor, create new ERDs if they
	// 		don't exist in governor.
	createERDctx, createERDSpan := s.tracer.Start(ctx, "create-erds")
	defer createERDSpan.End()

	for _, erd := range localERDs {
		if _, ok := govERDsSet[erd.SlugSingular]; ok {
			s.logger.Debug("ERD already exists, skipping", zap.String("slug", erd.SlugSingular))
			continue
		}

		s.logger.Debug("creating ERD", zap.String("slug", erd.SlugSingular))

		_, err := s.governorClient.CreateExtensionResourceDefinition(createERDctx, s.extensionID, erd)
		if err != nil {
			createERDSpan.SetStatus(codes.Error, err.Error())
			s.logger.Error(
				"failed creating ERD, this ERD will not be supported",
				zap.Error(err), zap.String("slug", erd.SlugSingular),
			)
		}
	}

	createERDSpan.End()

	// register processors
	for _, processor := range s.processors {
		processor.Register(s.eventRouter, s.extension)
	}

	return nil
}

// listERDsFromGovernor is a helper function that lists ERDs from governor
// and returns a map of ERD singular slugs
func listERDsFromGovernor(ctx context.Context, governorClient *govclient.Client, extensionID string) (map[string]byte, error) {
	erds, err := governorClient.ExtensionResourceDefinitions(ctx, extensionID, false)
	if err != nil {
		return nil, err
	}

	govERDsSet := make(map[string]byte, len(erds))
	for _, erd := range erds {
		govERDsSet[erd.SlugSingular] = 0
	}

	return govERDsSet, nil
}

// readERDsFromLocalDir is a helper function that reads ERDs from a local directory
func (s *Server) readERDsFromLocalDir(ctx context.Context, erdDir string) ([]*v1alpha1.ExtensionResourceDefinitionReq, error) {
	ctx, span := s.tracer.Start(ctx, "read-erds-from-local-dir")
	defer span.End()

	files, err := os.ReadDir(erdDir)
	if err != nil {
		return nil, err
	}

	localERDs := make([]*v1alpha1.ExtensionResourceDefinitionReq, 0, len(files))

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		path := filepath.Join(erdDir, file.Name())
		ext := filepath.Ext(path)

		bytes, err := os.ReadFile(path)
		if err != nil {
			s.logger.Warn("failed reading file", zap.Error(err), zap.String("file", path))
			continue
		}

		var contents erdvalidator.ERDContent

		_, span := s.tracer.Start(ctx, "read file", trace.WithAttributes(
			attribute.String("file", path),
		))
		defer span.End()

		switch ext {
		case ".json":
			contents = (*erdvalidator.ERDContentJSON)(&bytes)
		case ".yaml", ".yml":
			contents = (*erdvalidator.ERDContentYAML)(&bytes)
		default:
			s.logger.Warn("file type not supported", zap.String("file", path), zap.String("file-extension", ext))
			continue
		}

		erd, err := contents.Unmarshal()
		if err != nil {
			s.logger.Warn("failed unmarshalling ERD", zap.Error(err), zap.String("file", path))
			continue
		}

		localERDs = append(localERDs, erd)
	}

	return localERDs, nil
}
