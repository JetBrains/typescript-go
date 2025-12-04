package lsp

import (
	"context"
	"errors"
	"runtime/debug"

	"github.com/microsoft/typescript-go/internal/collections"
	"github.com/microsoft/typescript-go/internal/lsp/lsproto"
	"github.com/microsoft/typescript-go/internal/project"
)

var ProjectNotFoundError = errors.New("ProjectNotFoundError")

func (s *Server) handleCustomTsServerCommand(ctx context.Context, req *lsproto.RequestMessage) error {

	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			s.Log("panic running handleCustomTsServerCommand:", r, string(stack))
			s.sendResult(req.ID, &map[string]string{})
		}
	}()

	params := req.Params.(*lsproto.HandleCustomTsServerCommandParams)
	switch params.IdeCommand {
	case lsproto.IdeCommandGetElementType:
		{
			args := params.Arguments.(*lsproto.GetElementTypeArguments)
			project, file, err := s.GetProjectAndFileName(args.ProjectFileName, args.File, ctx)
			if err != nil {
				s.sendCustomCmdResult(req.ID, nil, err)
				return nil
			}

			element, err := IdeGetTypeOfElement(ctx, project, file, &args.Range, args.ForceReturnType, args.TypeRequestKind)
			s.sendCustomCmdResult(req.ID, element, err)
		}
	case lsproto.IdeCommandGetSymbolType:
		{
			args := params.Arguments.(*lsproto.GetSymbolTypeArguments)
			symbolType, err := IdeGetSymbolType(ctx, args.IdeProjectId, uint64(args.IdeTypeCheckerId), args.SymbolId)
			s.sendCustomCmdResult(req.ID, symbolType, err)
		}
	case lsproto.IdeCommandGetTypeProperties:
		{
			args := params.Arguments.(*lsproto.GetTypePropertiesArguments)
			typeProperties, err := IdeGetTypeProperties(ctx, args.IdeProjectId, uint64(args.IdeTypeCheckerId), args.TypeId)
			s.sendCustomCmdResult(req.ID, typeProperties, err)
		}
	case lsproto.IdeCommandGetTypeProperty:
		{
			args := params.Arguments.(*lsproto.GetTypePropertyArguments)
			symbol, err := IdeGetTypeProperty(ctx, args.IdeProjectId, uint64(args.IdeTypeCheckerId), args.TypeId, args.PropertyName)
			s.sendCustomCmdResult(req.ID, symbol, err)
		}

	case lsproto.IdeAreTypesMutuallyAssignable:
		{
			args := params.Arguments.(*lsproto.AreTypesMutuallyAssignableArguments)
			result, err := AreTypesMutuallyAssignable(ctx, args.IdeProjectId, uint64(args.IdeTypeCheckerId), args.Type1Id, args.Type2Id)
			s.sendCustomCmdResult(req.ID, result, err)
		}
	case lsproto.IdeGetResolvedSignature:
		{
			args := params.Arguments.(*lsproto.GetResolvedSignatureArguments)
			project, file, err := s.GetProjectAndFileName(args.ProjectFileName, args.File, ctx)
			if err != nil {
				s.sendCustomCmdResult(req.ID, nil, err)
				return nil
			}

			result, err := GetResolvedSignature(ctx, project, file, args.Range)
			s.sendCustomCmdResult(req.ID, result, err)
		}
	}
	snapshot, release := s.session.Snapshot()
	defer release()

	CleanupProjectsCache(append(snapshot.ProjectCollection.Projects(), GetAllSelfManagedProjects(s, ctx)...), s.logger)
	return nil
}

func (s *Server) GetProjectAndFileName(
	projectFileNameUri *lsproto.DocumentUri,
	fileUri lsproto.DocumentUri,
	ctx context.Context,
) (*project.Project, string, error) {
	file := fileUri.FileName()

	snapshot, release := s.session.Snapshot()
	released := false
	releaseOnce := func() {
		if !released {
			release()
			released = true
		}
	}
	defer releaseOnce()

	if projectFileNameUri != nil {
		projectFileName := projectFileNameUri.FileName()

		if IsSelfManagedProject(projectFileName) {
			if p := GetOrCreateSelfManagedProjectForFile(s, projectFileName, file, ctx); p != nil {
				return p, file, nil
			}
		}

		for _, p := range snapshot.ProjectCollection.Projects() {
			if p.Name() == projectFileName && p.GetProgram().GetSourceFile(file) != nil {
				return p, file, nil
			}
		}

		if p := GetOrCreateSelfManagedProjectForFile(s, projectFileName, file, ctx); p != nil {
			return p, file, nil
		}
	}

	if p := snapshot.GetDefaultProject(fileUri); p != nil {
		return p, file, nil
	}

	releaseOnce()

	if _, err := s.session.GetLanguageService(ctx, fileUri); err == nil {
		// Get a fresh snapshot since GetLanguageService may have updated it
		newSnapshot, release := s.session.Snapshot()
		defer release()
		if p := newSnapshot.GetDefaultProject(fileUri); p != nil {
			return p, file, nil
		}
	}

	// No project found
	return nil, file, ProjectNotFoundError
}

func (s *Server) sendCustomCmdResult(id *lsproto.ID, result *collections.OrderedMap[string, interface{}], err error) {
	response := make(map[string]interface{})
	if err == nil {
		response["response"] = result
	} else {
		errorResponse := make(map[string]interface{})
		errorResponse["error"] = err.Error()
		response["response"] = errorResponse
	}
	s.sendResult(id, response)
}
