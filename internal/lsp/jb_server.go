// MODIFIED IN THIS FORK: 2025-09-12 Konstantin Ulitin e3893948a2061ee66b3bc9e3df22590aba344872 support ideGetTypeProperty command
// MODIFIED IN THIS FORK: 2025-08-25 Aleksei Berezkin 8aa1bd88535b085be08c607ccbecbe4e6fbfeeea Fixed fork after refactoring in upstream
// MODIFIED IN THIS FORK: 2025-08-14 Aleksei Berezkin 57b2402e912f23f4a1345039460fac83895ce6c8 WEB-74070 Migrated away from private defaultProjectFinder.findOrCreateProject
// MODIFIED IN THIS FORK: 2025-08-05 Aleksei Berezkin a9368494725330485924ed5ff3db447110dab0b3 WEB-74070 Projects cache cleanup
// MODIFIED IN THIS FORK: 2025-07-25 Aleksei Berezkin 1b4d405531446e2f049414861843068ad508d68d WEB-74070 Do not reopen a file if it's already opened
// MODIFIED IN THIS FORK: 2025-07-24 Aleksei Berezkin 5567c01219bf93fad344a80691da5df021923d10 WEB-74070 Supported the method IdeGetResolvedSignature
// MODIFIED IN THIS FORK: 2025-05-21 Piotr Tomiak dfd337df3e64d6fb2bc18c46bf4de05bcbea5a64 Support WebStorm types in LSP mode
package lsp

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/microsoft/typescript-go/internal/collections"
	"github.com/microsoft/typescript-go/internal/jsonrpc"
	"github.com/microsoft/typescript-go/internal/lsp/lsproto"
	"github.com/microsoft/typescript-go/internal/project"
)

var ProjectNotFoundError = errors.New("ProjectNotFoundError")

func (s *Server) jbHandleCustomTsServerCommand(ctx context.Context, req *lsproto.RequestMessage) error {
	// !!! most likely not needed once support is fully implemented
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			s.logger.Error("panic running jbHandleCustomTsServerCommand:", r, string(stack))
			s.sendResult(req.ID, &map[string]string{})
		}
	}()
	params := req.Params.(*lsproto.JbHandleCustomTsServerCommandParams)
	switch params.IdeCommand {
	case lsproto.IdeCommandGetElementType:
		{
			args := params.Arguments.(*lsproto.GetElementTypeArguments)
			project, file, err := s.GetProjectAndFileName(args.ProjectFileName, args.File, ctx)
			if err != nil {
				s.jbSendResult(req.ID, nil, err)
				return nil
			}

			element, err := IdeGetTypeOfElement(ctx, project, file, &args.Range, args.ForceReturnType, args.TypeRequestKind)
			s.jbSendResult(req.ID, element, err)
		}
	case lsproto.IdeCommandGetSymbolType:
		{
			args := params.Arguments.(*lsproto.GetSymbolTypeArguments)
			symbolType, err := IdeGetSymbolType(ctx, args.IdeProjectId, uint64(args.IdeTypeCheckerId), args.SymbolId)
			s.jbSendResult(req.ID, symbolType, err)
		}
	case lsproto.IdeCommandGetTypeProperties:
		{
			args := params.Arguments.(*lsproto.GetTypePropertiesArguments)
			typeProperties, err := IdeGetTypeProperties(ctx, args.IdeProjectId, uint64(args.IdeTypeCheckerId), args.TypeId)
			s.jbSendResult(req.ID, typeProperties, err)
		}
	case lsproto.IdeCommandGetTypeProperty:
		{
			args := params.Arguments.(*lsproto.GetTypePropertyArguments)
			symbol, err := IdeGetTypeProperty(ctx, args.IdeProjectId, uint64(args.IdeTypeCheckerId), args.TypeId, args.PropertyName)
			s.jbSendResult(req.ID, symbol, err)
		}
	case lsproto.IdeCommandGetTypeText:
		{
			args := params.Arguments.(*lsproto.GetTypeTextArguments)
			typeText, err := IdeGetTypeText(ctx, args.IdeProjectId, uint64(args.IdeTypeCheckerId), args.TypeId, args.SymbolId, args.Flags)
			s.jbSendResult(req.ID, typeText, err)
		}

	case lsproto.IdeAreTypesMutuallyAssignable:
		{
			args := params.Arguments.(*lsproto.AreTypesMutuallyAssignableArguments)
			result, err := AreTypesMutuallyAssignable(ctx, args.IdeProjectId, uint64(args.IdeTypeCheckerId), args.Type1Id, args.Type2Id)
			s.jbSendResult(req.ID, result, err)
		}
	case lsproto.IdeGetResolvedSignature:
		{
			args := params.Arguments.(*lsproto.GetResolvedSignatureArguments)
			project, file, err := s.GetProjectAndFileName(args.ProjectFileName, args.File, ctx)
			if err != nil {
				s.jbSendResult(req.ID, nil, err)
				return nil
			}

			result, err := GetResolvedSignature(ctx, project, file, args.Range)
			s.jbSendResult(req.ID, result, err)
		}
	case lsproto.IdeCommandGetCompletionWithSymbols:
		{
			args := params.Arguments.(*lsproto.GetCompletionSymbolsArguments)
			proj, file, err := s.GetProjectAndFileName(args.ProjectFileName, args.File, ctx)
			if err != nil {
				s.jbSendResult(req.ID, nil, err)
				return nil
			}

			snapshot := s.session.Snapshot()
			result, err := IdeGetCompletionWithSymbols(ctx, proj, snapshot, file, args.Position)
			s.jbSendResult(req.ID, result, err)
		}
	}
	snapshot := s.session.Snapshot()

	CleanupProjectsCache(append(snapshot.ProjectCollection.Projects(), GetAllSelfManagedProjects(s, ctx)...), s.logger)
	return nil
}

func (s *Server) GetProjectAndFileName(
	projectFileNameUri *lsproto.DocumentUri,
	fileUri lsproto.DocumentUri,
	ctx context.Context,
) (*project.Project, string, error) {
	file, err := documentURIFileName(fileUri)
	if err != nil {
		return nil, "", err
	}

	var projectFileName string
	if projectFileNameUri != nil {
		projectFileName, err = documentURIFileName(*projectFileNameUri)
		if err != nil {
			return nil, file, fmt.Errorf("invalid projectFileName: %w", err)
		}

		if IsSelfManagedProject(projectFileName) {
			if p := GetOrCreateSelfManagedProjectForFile(s, projectFileName, file, ctx); p != nil {
				return p, file, nil
			}
		}
	}

	_, snapshot, err := s.session.GetLanguageServiceAndSnapshot(ctx, fileUri)
	if err != nil {
		if projectFileNameUri != nil && canOpenSelfManagedProject(s, projectFileName) {
			if p := GetOrCreateSelfManagedProjectForFile(s, projectFileName, file, ctx); p != nil {
				return p, file, nil
			}
		}

		return nil, file, ProjectNotFoundError
	}

	if projectFileNameUri != nil {
		for _, p := range snapshot.ProjectCollection.Projects() {
			if p.Name() == projectFileName && p.GetProgram().GetSourceFile(file) != nil {
				return p, file, nil
			}
		}

		if canOpenSelfManagedProject(s, projectFileName) {
			if p := GetOrCreateSelfManagedProjectForFile(s, projectFileName, file, ctx); p != nil {
				return p, file, nil
			}
		}
	}

	if p := snapshot.GetDefaultProject(fileUri); p != nil {
		return p, file, nil
	}

	// No project found
	return nil, file, ProjectNotFoundError

	/*
		// Unstable API, skip so far

		canonicalFileName := tspath.GetCanonicalFileName(file, s.projectService.FS().UseCaseSensitiveFileNames())
		scriptInfo := s.projectService.DocumentStore().GetScriptInfoByPath(tspath.Path(canonicalFileName))
		if scriptInfo != nil {
			return scriptInfo.ContainingProjects()[0], file
		}

		// Default project - may be inferred
		// TODO - handle list of automatically opened files and close them automatically as well
		fileContents, ok := s.projectService.FS().ReadFile(file)
		if !ok {
			panic("Failed to read " + file)
		}
		scriptKind := core.GetScriptKindFromFileName(file)
		s.projectService.OpenFile(file, fileContents, scriptKind, file)
		_, proj = s.projectService.EnsureDefaultProjectForFile(file)

		return proj, file
	*/
}

func documentURIFileName(uri lsproto.DocumentUri) (fileName string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("invalid URI %q: %v", uri, r)
		}
	}()

	return uri.FileName(), nil
}

func canOpenSelfManagedProject(s *Server, projectFileName string) bool {
	if s.fs.FileExists(projectFileName) {
		return true
	}

	s.logger.Logf("SelfManagedProjects:: Skipping project %s because the project file does not exist", projectFileName)
	return false
}

func (s *Server) jbSendResult(id *jsonrpc.ID, result *collections.OrderedMap[string, interface{}], err error) {
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
