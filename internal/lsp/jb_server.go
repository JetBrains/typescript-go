package lsp

import (
	"context"
	"runtime/debug"

	"github.com/microsoft/typescript-go/internal/collections"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/ls"
	"github.com/microsoft/typescript-go/internal/lsp/lsproto"
	"github.com/microsoft/typescript-go/internal/project"
	"github.com/microsoft/typescript-go/internal/tspath"
)

func (s *Server) jbHandleCustomTsServerCommand(ctx context.Context, req *lsproto.RequestMessage) error {
	// !!! most likely not needed once support is fully implemented
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			s.Log("panic running jbHandleCustomTsServerCommand:", r, string(stack))
			s.sendResult(req.ID, &map[string]string{})
		}
	}()
	params := req.Params.(*lsproto.JbHandleCustomTsServerCommandParams)
	switch params.IdeCommand {
	case lsproto.IdeCommandGetElementType:
		{
			args := params.Arguments.(*lsproto.GetElementTypeArguments)
			project, file := s.GetProjectAndFileName(args.ProjectFileName, args.File)
			element, err := IdeGetTypeOfElement(ctx, project, file, &args.Range, args.ForceReturnType, args.TypeRequestKind)
			s.jbSendResult(req.ID, element, err)
		}
	case lsproto.IdeCommandGetSymbolType:
		{
			args := params.Arguments.(*lsproto.GetSymbolTypeArguments)
			symbolType, err := IdeGetSymbolType(ctx, args.IdeProjectId, args.IdeTypeCheckerId, args.SymbolId)
			s.jbSendResult(req.ID, symbolType, err)
		}
	case lsproto.IdeCommandGetTypeProperties:
		{
			args := params.Arguments.(*lsproto.GetTypePropertiesArguments)
			typeProperties, err := IdeGetTypeProperties(ctx, args.IdeProjectId, args.IdeTypeCheckerId, args.TypeId)
			s.jbSendResult(req.ID, typeProperties, err)
		}

	case lsproto.IdeAreTypesMutuallyAssignable:
		{
			args := params.Arguments.(*lsproto.AreTypesMutuallyAssignableArguments)
			result, err := AreTypesMutuallyAssignable(ctx, args.IdeProjectId, args.IdeTypeCheckerId, args.Type1Id, args.Type2Id)
			s.jbSendResult(req.ID, result, err)
		}
	case lsproto.IdeGetResolvedSignature:
		{
			args := params.Arguments.(*lsproto.GetResolvedSignatureArguments)
			project, file := s.GetProjectAndFileName(args.ProjectFileName, args.File)

			result, err := GetResolvedSignature(ctx, project, file, args.Range)
			s.jbSendResult(req.ID, result, err)
		}
	}
	CleanupProjectsCache(s.projectService.Projects())
	return nil
}

func (s *Server) GetProjectAndFileName(projectFileName *lsproto.DocumentUri, fileUri lsproto.DocumentUri) (*project.Project, string) {
	var project *project.Project
	file := ls.DocumentURIToFileName(fileUri)

	if projectFileName != nil {
		projectFileName := ls.DocumentURIToFileName(*projectFileName)
		project = s.projectService.FindOrCreateConfiguredProject(projectFileName, true)
		if project.GetProgram().GetSourceFile(file) == nil {
			// load a default project for the file
			project = nil
		}
	}

	if project == nil {
		canonicalFileName := tspath.GetCanonicalFileName(file, s.projectService.FS().UseCaseSensitiveFileNames())
		scriptInfo := s.projectService.DocumentStore().GetScriptInfoByPath(tspath.Path(canonicalFileName))
		if scriptInfo != nil {
			return scriptInfo.ContainingProjects()[0], file
		}

		fileContents, ok := s.projectService.FS().ReadFile(file)
		if !ok {
			panic("Failed to read " + file)
		}
		scriptKind := core.GetScriptKindFromFileName(file)
		// TODO - handle list of automatically opened files and close them automatically as well
		s.projectService.OpenFile(file, fileContents, scriptKind, file)
		_, project = s.projectService.EnsureDefaultProjectForFile(file)
	}

	return project, file
}

func (s *Server) jbSendResult(id *lsproto.ID, result *collections.OrderedMap[string, interface{}], err error) {
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
