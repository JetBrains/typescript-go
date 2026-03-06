// MODIFIED IN THIS FORK: 2025-09-12 Konstantin Ulitin e3893948a2061ee66b3bc9e3df22590aba344872 support ideGetTypeProperty command
// MODIFIED IN THIS FORK: 2025-09-08 Aleksei Berezkin 6dc83473b479cc13fb5618ce8498ba95855ebbcc Muted SourceFileNotFoundError -- former WS next just returns null here
// MODIFIED IN THIS FORK: 2025-08-25 Aleksei Berezkin 8aa1bd88535b085be08c607ccbecbe4e6fbfeeea Fixed fork after refactoring in upstream
// MODIFIED IN THIS FORK: 2025-08-07 Aleksei Berezkin 2c962510b51eba462c525d251e5400ed00078424 WEB-74070 Fixed unicode serialization
// MODIFIED IN THIS FORK: 2025-08-06 Aleksei Berezkin 83dd6da40d94b76c6dd194f2abd8b6638d28f335 WEB-74070 Comment cleanup
// MODIFIED IN THIS FORK: 2025-08-05 Aleksei Berezkin a9368494725330485924ed5ff3db447110dab0b3 WEB-74070 Projects cache cleanup
// MODIFIED IN THIS FORK: 2025-07-30 Aleksei Berezkin 71fcb2670191ea30ff01d91bbe90b247622bd63a WEB-74070 Fixed getting type of type reference
// MODIFIED IN THIS FORK: 2025-07-24 Aleksei Berezkin dc01ba3ed67541c5b322d4593a419fed1595c6ee WEB-74070 Fixed computed property conversion
// MODIFIED IN THIS FORK: 2025-07-24 Aleksei Berezkin 5567c01219bf93fad344a80691da5df021923d10 WEB-74070 Supported the method IdeGetResolvedSignature
// MODIFIED IN THIS FORK: 2025-07-23 Aleksei Berezkin 34f47754ded347ce69f74fe5c12289ac132eec88 WEB-74070 Fixed properties conversion
// MODIFIED IN THIS FORK: 2025-07-22 Aleksei Berezkin 6f522164b5f2df9cfd7bab60d843a11ef6d31555 WEB-74070 Fixed converting generic type args
// MODIFIED IN THIS FORK: 2025-05-29 Andrey Vorobev c2c9d57fdce88db9c175086876a86721938fe679 [js] WEB-72440 No attribute props from motion library with enabled service type engine - update TS Go
// MODIFIED IN THIS FORK: 2025-05-21 Piotr Tomiak dfd337df3e64d6fb2bc18c46bf4de05bcbea5a64 Support WebStorm types in LSP mode
package lsp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/astnav"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/collections"
	"github.com/microsoft/typescript-go/internal/jsnum"
	"github.com/microsoft/typescript-go/internal/ls"
	"github.com/microsoft/typescript-go/internal/lsp/lsproto"
	"github.com/microsoft/typescript-go/internal/project"
	"github.com/microsoft/typescript-go/internal/project/logging"
	"github.com/microsoft/typescript-go/internal/scanner"
)

var (
	//nolint
	OutdatedProjectVersionError = errors.New("OutdatedTypeCheckerIdException")
	//nolint
	OutdatedProjectIdError = errors.New("OutdatedProjectIdException")
	//nolint
	// SourceFileNotFoundError = errors.New("SourceFileNotFoundError")
	// TODO When sourceFile not found, we're returning nil just like in former WS next
	// TODO However it probably should be handled
)

var (
	nextProjectId   = 1
	project2IdMap   = make(map[*project.Project]int)
	id2ProjectMap   = make(map[int]*project.Project)
	projectMapMutex = &sync.Mutex{}
	projectCacheMap = make(map[int]*ProjectCache)
)

type ProjectCache struct {
	projectVersion   uint64
	requestedTypeIds *collections.Set[checker.TypeId]
	seenTypeIds      map[checker.TypeId]*checker.Type
	seenSymbolIds    map[ast.SymbolId]*ast.Symbol
}

func getProjectId(project *project.Project) int {
	projectMapMutex.Lock()
	defer projectMapMutex.Unlock()
	result, ok := project2IdMap[project]
	if !ok {
		result = nextProjectId
		nextProjectId++
		project2IdMap[project] = result
		id2ProjectMap[result] = project
	}
	return result
}

func getProject(projectId int) (*project.Project, bool) {
	projectMapMutex.Lock()
	defer projectMapMutex.Unlock()
	id, ok := id2ProjectMap[projectId]
	return id, ok
}

func getProjectCache(projectId int, projectVersion uint64) *ProjectCache {
	projectMapMutex.Lock()
	defer projectMapMutex.Unlock()
	result, ok := projectCacheMap[projectId]
	if !ok || result.projectVersion != projectVersion {
		result = &ProjectCache{
			projectVersion:   projectVersion,
			requestedTypeIds: collections.NewSetWithSizeHint[checker.TypeId](10),
			seenTypeIds:      make(map[checker.TypeId]*checker.Type),
			seenSymbolIds:    make(map[ast.SymbolId]*ast.Symbol),
		}
		projectCacheMap[projectId] = result
	}
	return result
}

func CleanupProjectsCache(openedProjects []*project.Project, logger logging.Logger) {
	openedProjectsSet := collections.NewSetWithSizeHint[*project.Project](len(openedProjects))
	for _, p := range openedProjects {
		if p != nil {
			openedProjectsSet.Add(p)
		}
	}

	removedIds := cleanupProjectsCacheImpl(openedProjectsSet)

	if removedIds.Len() > 0 && len(openedProjects) > 0 && openedProjects[0] != nil {
		ids := make([]int, 0, removedIds.Len())
		for id := range removedIds.Keys() {
			ids = append(ids, id)
		}
		logger.Logf("CleanupProjectsCache: removed project ids: %v", ids)
	}
}

func cleanupProjectsCacheImpl(openedProjectsSet *collections.Set[*project.Project]) *collections.Set[int] {
	projectMapMutex.Lock()
	defer projectMapMutex.Unlock()

	removedIds := collections.NewSetWithSizeHint[int](len(id2ProjectMap) + len(projectCacheMap))

	for p := range project2IdMap {
		if !openedProjectsSet.Has(p) {
			delete(project2IdMap, p)
		}
	}

	validProjectIdsSet := collections.NewSetWithSizeHint[int](len(project2IdMap))
	for _, id := range project2IdMap {
		validProjectIdsSet.Add(id)
	}

	for id := range id2ProjectMap {
		if !validProjectIdsSet.Has(id) {
			delete(id2ProjectMap, id)
			removedIds.Add(id)
		}
	}

	for id := range projectCacheMap {
		if !validProjectIdsSet.Has(id) {
			delete(projectCacheMap, id)
			removedIds.Add(id)
		}
	}

	return removedIds
}

func IdeGetTypeOfElement(
	ctx context.Context,
	project *project.Project,
	fileName string,
	Range *lsproto.Range,
	forceReturnType bool,
	typeRequestKind lsproto.TypeRequestKind,
) (*collections.OrderedMap[string, interface{}], error) {
	projectIdNum := getProjectId(project)
	projectVersion := project.ProgramLastUpdate // TODO: possible race condition here
	program := project.GetProgram()
	sourceFile := program.GetSourceFile(fileName)
	if sourceFile == nil {
		return nil, nil
	}

	startOffset := scanner.GetECMAPositionOfLineAndCharacter(sourceFile, int(Range.Start.Line), int(Range.Start.Character))
	endOffset := scanner.GetECMAPositionOfLineAndCharacter(sourceFile, int(Range.End.Line), int(Range.End.Character))

	node := astnav.GetTokenAtPosition(sourceFile, startOffset).AsNode()
	for node != nil && node.End() < endOffset {
		node = node.Parent
	}
	if node == nil || node == sourceFile.AsNode() {
		return nil, nil
	}

	contextFlags := typeRequestKindToContextFlags(typeRequestKind)
	isContextual := contextFlags >= 0
	if isContextual && !ast.IsExpression(node) || (!isContextual && (ast.IsStringLiteral(node) || ast.IsNumericLiteral(node))) || ast.IsIdentifier(node) && ast.IsTypeReferenceNode(node.Parent) {
		if node.Pos() == node.Parent.Pos() && node.End() == node.Parent.End() {
			node = node.Parent
		}
	}

	typeChecker, done := program.GetTypeCheckerForFile(ctx, sourceFile)
	defer done()

	var t *checker.Type
	if contextFlags >= 0 {
		t = typeChecker.GetContextualType(node, checker.ContextFlags(contextFlags))
	} else {
		t = typeChecker.GetTypeAtLocation(node)
	}

	if t == nil && isContextual && node.Parent.Kind == ast.KindBinaryExpression {
		parentBinaryExpr := node.Parent.AsBinaryExpression()
		// from getContextualType in services/completions.ts
		right := parentBinaryExpr.Right
		if ls.IsEqualityOperatorKind(parentBinaryExpr.OperatorToken.Kind) {
			if node == right {
				t = typeChecker.GetTypeAtLocation(parentBinaryExpr.Left)
			} else {
				t = typeChecker.GetTypeAtLocation(right)
			}
		}
	}
	if t == nil {
		return nil, nil
	}

	convertContext := NewConvertContext(typeChecker, projectIdNum, projectVersion)

	var typeMap *collections.OrderedMap[string, interface{}]
	if forceReturnType || !convertContext.requestedTypeIds.Has(t.Id()) {
		typeMap = ConvertType(t, convertContext)
		convertContext.requestedTypeIds.Add(t.Id())
	} else {
		typeMap = collections.NewOrderedMapWithSizeHint[string, interface{}](4)
		typeMap.Set("id", t.Id())
		typeMap.Set("ideObjectType", "TypeObject")
	}
	typeMap.Set("ideProjectId", projectIdNum)
	typeMap.Set("ideTypeCheckerId", projectVersion)
	return typeMap, nil
}

func getConvertContext(
	ctx context.Context,
	projectId int,
	projectVersion uint64,
) (*ConvertContext, func(), error) {
	project, ok := getProject(projectId)
	if !ok {
		return nil, func() {}, OutdatedProjectIdError
	}

	if projectVersion != project.ProgramLastUpdate {
		return nil, func() {}, OutdatedProjectVersionError
	}
	checker, done := project.GetProgram().GetTypeChecker(ctx)
	return NewConvertContext(checker, projectId, projectVersion), func() {
		done()
	}, nil
}

func IdeGetSymbolType(
	ctx context.Context,
	projectId int,
	projectVersion uint64,
	symbolId int,
) (*collections.OrderedMap[string, interface{}], error) {
	convertContext, done, err := getConvertContext(ctx, projectId, projectVersion)
	defer done()
	if err != nil {
		return nil, err
	}

	symbol, exists := convertContext.seenSymbolIds[ast.SymbolId(symbolId)]
	if !exists {
		return nil, errors.New("symbol not found")
	}

	t := convertContext.checker.GetTypeOfSymbol(symbol)
	result := ConvertType(t, convertContext)
	result.Set("ideProjectId", projectId)
	result.Set("ideTypeCheckerId", projectVersion)
	return result, nil
}

func IdeGetTypeProperties(
	ctx context.Context,
	projectId int,
	projectVersion uint64,
	typeId int,
) (*collections.OrderedMap[string, interface{}], error) {
	convertContext, done, err := getConvertContext(ctx, projectId, projectVersion)
	defer done()
	if err != nil {
		return nil, err
	}

	t, exists := convertContext.seenTypeIds[checker.TypeId(typeId)]
	if !exists {
		return nil, errors.New("type not found")
	}

	result := ConvertTypeProperties(t, convertContext)
	result.Set("ideProjectId", projectId)
	result.Set("ideTypeCheckerId", projectVersion)
	return result, nil
}

func IdeGetTypeProperty(
	ctx context.Context,
	projectId int,
	projectVersion uint64,
	typeId int,
	propertyName string,
) (*collections.OrderedMap[string, interface{}], error) {
	convertContext, done, err := getConvertContext(ctx, projectId, projectVersion)
	defer done()
	if err != nil {
		return nil, err
	}

	t, exists := convertContext.seenTypeIds[checker.TypeId(typeId)]
	if !exists {
		return nil, errors.New("type not found")
	}

	symbol := convertContext.checker.GetPropertyOfType(t, propertyName)
	if symbol == nil {
		return nil, nil
	}
	result := ConvertSymbol(symbol, convertContext)
	return result, nil
}

func IdeGetTypeText(
	ctx context.Context,
	projectId int,
	projectVersion uint64,
	typeId *int,
	symbolId *int,
	flags *int,
) (*collections.OrderedMap[string, interface{}], error) {
	convertContext, done, err := getConvertContext(ctx, projectId, projectVersion)
	defer done()
	if err != nil {
		return nil, err
	}

	var t *checker.Type
	var symbol *ast.Symbol

	if typeId != nil {
		var exists bool
		t, exists = convertContext.seenTypeIds[checker.TypeId(*typeId)]
		if !exists {
			return nil, nil
		}
		symbol = t.Symbol()
	} else if symbolId != nil {
		var exists bool
		symbol, exists = convertContext.seenSymbolIds[ast.SymbolId(*symbolId)]
		if !exists {
			return nil, nil
		}
		t = convertContext.checker.GetTypeOfSymbol(symbol)
	}

	if t == nil {
		return nil, nil
	}

	var formatFlags checker.TypeFormatFlags
	if flags != nil {
		formatFlags = checker.TypeFormatFlags(*flags)
	} else {
		formatFlags = checker.TypeFormatFlagsAllowUniqueESSymbolType | checker.TypeFormatFlagsUseAliasDefinedOutsideCurrentScope
	}

	var enclosingDeclaration *ast.Node
	if symbol != nil && symbol.Declarations != nil && len(symbol.Declarations) == 1 {
		enclosingDeclaration = symbol.Declarations[0]
	}
	typeText := convertContext.checker.TypeToStringEx(t, enclosingDeclaration, formatFlags)

	result := collections.NewOrderedMapWithSizeHint[string, interface{}](1)
	result.Set("typeText", typeText)
	return result, nil
}

func IdeGetCompletionWithSymbols(
	ctx context.Context,
	proj *project.Project,
	snapshot *project.Snapshot,
	fileName string,
	position lsproto.Position,
) (*collections.OrderedMap[string, interface{}], error) {
	projectIdNum := getProjectId(proj)
	projectVersion := proj.ProgramLastUpdate
	program := proj.GetProgram()
	sourceFile := program.GetSourceFile(fileName)
	if sourceFile == nil {
		return nil, nil
	}

	positionOffset := scanner.GetECMAPositionOfLineAndCharacter(sourceFile, int(position.Line), int(position.Character))

	languageService := ls.NewLanguageService(proj.ConfigFilePath(), program, snapshot)

	typeCheckerForData, doneForData := languageService.GetProgram().GetTypeCheckerForFile(ctx, sourceFile)
	defer doneForData()
	preferences := languageService.UserPreferences()
	completionInfo := languageService.GetCompletionInfoWithSymbols(ctx, typeCheckerForData, sourceFile, positionOffset, preferences)

	if completionInfo == nil {
		return nil, nil
	}

	typeChecker, done := program.GetTypeCheckerForFile(ctx, sourceFile)
	defer done()

	convertContext := NewConvertContext(typeChecker, projectIdNum, projectVersion)

	convertedSymbols := make([]interface{}, 0, len(completionInfo.Symbols))
	for _, symbol := range completionInfo.Symbols {
		convertedSymbols = append(convertedSymbols, ConvertSymbol(symbol, convertContext))
	}

	result := collections.NewOrderedMapWithSizeHint[string, interface{}](3)
	result.Set("ideProjectId", projectIdNum)
	result.Set("ideTypeCheckerId", projectVersion)
	result.Set("items", completionInfo.Items)
	result.Set("symbols", convertedSymbols)
	return result, nil
}

func AreTypesMutuallyAssignable(
	ctx context.Context,
	projectId int,
	projectVersion uint64,
	type1Id int,
	type2Id int,
) (*collections.OrderedMap[string, interface{}], error) {
	convertCtx, done, err := getConvertContext(ctx, projectId, projectVersion)
	defer done()
	if err != nil {
		return nil, err
	}

	type1, exists := convertCtx.seenTypeIds[checker.TypeId(type1Id)]
	if !exists {
		return nil, errors.New("type1 not found")
	}

	type2, exists := convertCtx.seenTypeIds[checker.TypeId(type2Id)]
	if !exists {
		return nil, errors.New("type2 not found")
	}

	isType1To2 := convertCtx.checker.IsTypeAssignableTo(type1, type2)
	isType2To1 := convertCtx.checker.IsTypeAssignableTo(type2, type1)

	areMutuallyAssignable := isType1To2 && isType2To1

	result := collections.NewOrderedMapWithSizeHint[string, interface{}](2)
	result.Set("areMutuallyAssignable", areMutuallyAssignable)
	return result, nil
}

func GetResolvedSignature(
	ctx context.Context,
	project *project.Project,
	fileName string,
	Range lsproto.Range,
) (*collections.OrderedMap[string, interface{}], error) {
	projectId := getProjectId(project)
	projectVersion := project.ProgramLastUpdate
	program := project.Program
	sourceFile := program.GetSourceFile(fileName)
	if sourceFile == nil {
		return nil, nil
	}

	startOffset := scanner.GetECMAPositionOfLineAndCharacter(sourceFile, int(Range.Start.Line), int(Range.Start.Character))
	endOffset := scanner.GetECMAPositionOfLineAndCharacter(sourceFile, int(Range.End.Line), int(Range.End.Character))

	typeChecker, done := program.GetTypeCheckerForFile(ctx, sourceFile)
	defer done()

	// Find the node at the given position
	node := astnav.GetTokenAtPosition(sourceFile, startOffset).AsNode()
	for node != nil && node.End() < endOffset {
		node = node.Parent
	}

	if node == nil || node == sourceFile.AsNode() {
		return nil, nil
	}

	// Find the call expression
	for !ast.IsCallLikeExpression(node) {
		node = node.Parent
		if node == nil || node == sourceFile.AsNode() {
			return nil, nil
		}
	}

	// Get the resolved signature
	signature := typeChecker.GetResolvedSignature(node)
	if signature == nil {
		return nil, nil
	}

	// Return the signature information
	convertCtx := NewConvertContext(typeChecker, projectId, projectVersion)
	prepared := ConvertSignature(signature, convertCtx)
	prepared.Set("ideTypeCheckerId", projectVersion)
	prepared.Set("ideProjectId", projectId)
	prepared.Set("ideObjectType", "SignatureObject")

	return prepared, nil
}

type ConvertContext struct {
	nextId               int
	createdObjectsIdeIds map[interface{}]int
	checker              *checker.Checker
	requestedTypeIds     *collections.Set[checker.TypeId]
	seenTypeIds          map[checker.TypeId]*checker.Type
	seenSymbolIds        map[ast.SymbolId]*ast.Symbol
}

func NewConvertContext(checker *checker.Checker, projectId int, projectVersion uint64) *ConvertContext {
	cache := getProjectCache(projectId, projectVersion)
	return &ConvertContext{
		nextId:               0,
		createdObjectsIdeIds: make(map[interface{}]int),
		checker:              checker,
		requestedTypeIds:     cache.requestedTypeIds,
		seenTypeIds:          cache.seenTypeIds,
		seenSymbolIds:        cache.seenSymbolIds,
	}
}

func (ctx *ConvertContext) GetIdeObjectId(obj interface{}) (int, bool) {
	id, exists := ctx.createdObjectsIdeIds[obj]
	return id, exists
}

func (ctx *ConvertContext) RegisterIdeObject(obj interface{}) int {
	id := ctx.nextId
	ctx.nextId++
	ctx.createdObjectsIdeIds[obj] = id
	return id
}

func FindReferenceOrConvert(sourceObj interface{},
	convertTarget func(ideObjectId int) *collections.OrderedMap[string, interface{}], ctx *ConvertContext,
) *collections.OrderedMap[string, interface{}] {
	ideObjectId, exists := ctx.GetIdeObjectId(sourceObj)
	if exists {
		result := collections.NewOrderedMapWithSizeHint[string, interface{}](1)
		result.Set("ideObjectIdRef", ideObjectId)
		return result
	}

	ideObjectId = ctx.RegisterIdeObject(sourceObj)
	newObject := convertTarget(ideObjectId)
	return newObject
}

func ConvertType(t *checker.Type, ctx *ConvertContext) *collections.OrderedMap[string, interface{}] {
	return FindReferenceOrConvert(t, func(ideObjectId int) *collections.OrderedMap[string, interface{}] {
		tscType := collections.NewOrderedMapWithSizeHint[string, interface{}](15)
		tscType.Set("ideObjectId", ideObjectId)
		tscType.Set("ideObjectType", "TypeObject")
		tscType.Set("flags", convertTypeFlags(t.Flags()))

		// Handle aliasTypeArguments
		aliasType := t.Alias()
		if aliasType != nil && aliasType.TypeArguments() != nil {
			aliasArgs := make([]interface{}, 0)
			for _, t := range aliasType.TypeArguments() {
				aliasArgs = append(aliasArgs, ConvertType(t, ctx))
			}
			tscType.Set("aliasTypeArguments", aliasArgs)
		}

		if t.Flags()&checker.TypeFlagsObject != 0 {
			if target := t.Target(); target != nil {
				tscType.Set("target", ConvertType(target, ctx))
			}

			if (t.ObjectFlags() & checker.ObjectFlagsReference) != 0 {
				resolvedArgs := make([]interface{}, 0)
				typeArgs := ctx.checker.GetTypeArguments(t)
				for _, t := range typeArgs {
					// Filter out 'this' type
					if t.Flags()&checker.TypeFlagsTypeParameter == 0 || !t.AsTypeParameter().IsThisType() {
						resolvedArgs = append(resolvedArgs, ConvertType(t, ctx))
					}
				}
				tscType.Set("resolvedTypeArguments", resolvedArgs)
			}
		}

		// For UnionOrIntersection and TemplateLiteral types
		if t.Flags()&(checker.TypeFlagsUnionOrIntersection|checker.TypeFlagsTemplateLiteral) != 0 {
			typesArr := t.Types()
			types := make([]interface{}, 0)
			for _, t := range typesArr {
				types = append(types, ConvertType(t, ctx))
			}
			tscType.Set("types", types)
		}

		// For Literal types with freshType
		if t.Flags()&checker.TypeFlagsLiteral != 0 {
			literalType := t.AsLiteralType()
			if freshType := literalType.FreshType(); freshType != nil {
				tscType.Set("freshType", ConvertType(freshType, ctx))
			}
		}

		// For TypeParameter types
		if t.Flags()&checker.TypeFlagsTypeParameter != 0 {
			if constraint := ctx.checker.GetBaseConstraintOfType(t); constraint != nil {
				tscType.Set("constraint", ConvertType(constraint, ctx))
			}
		}

		// For Index types
		if t.Flags()&checker.TypeFlagsIndex != 0 {
			indexType := t.AsIndexType()
			tscType.Set("type", ConvertType(indexType.Target(), ctx))
		}

		// For IndexedAccess types
		if t.Flags()&checker.TypeFlagsIndexedAccess != 0 {
			indexedAccessType := t.AsIndexedAccessType()
			tscType.Set("objectType", ConvertType(indexedAccessType.ObjectType(), ctx))
			tscType.Set("indexType", ConvertType(indexedAccessType.IndexType(), ctx))
		}

		// For Conditional types
		if t.Flags()&checker.TypeFlagsConditional != 0 {
			conditionalType := t.AsConditionalType()
			tscType.Set("checkType", ConvertType(conditionalType.CheckType(), ctx))
			tscType.Set("extendsType", ConvertType(conditionalType.ExtendsType(), ctx))
		}

		// For Substitution types
		if t.Flags()&checker.TypeFlagsSubstitution != 0 {
			substitutionType := t.AsSubstitutionType()
			tscType.Set("baseType", ConvertType(substitutionType.BaseType(), ctx))
		}

		// Add symbol and aliasSymbol
		if t.Symbol() != nil {
			tscType.Set("symbol", ConvertSymbol(t.Symbol(), ctx))
		}
		if t.Alias() != nil && t.Alias().Symbol() != nil {
			tscType.Set("aliasSymbol", ConvertSymbol(t.Alias().Symbol(), ctx))
		}

		// Handle object flags
		if t.Flags()&checker.TypeFlagsObject != 0 {
			tscType.Set("objectFlags", convertObjectFlags(t.ObjectFlags()))
		}

		// Handle literal type values
		if t.Flags()&checker.TypeFlagsLiteral != 0 {
			literalType := t.AsLiteralType()
			if t.Flags()&checker.TypeFlagsBigIntLiteral != 0 {
				// Convert BigInt literal value
				tscType.Set("value", ConvertPseudoBigInt(literalType.Value().(jsnum.PseudoBigInt), ctx))
			} else {
				// For other literal types; skip non-finite numbers as they can't be marshaled to JSON
				v := literalType.Value()
				if n, ok := v.(jsnum.Number); ok && (n.IsInf() || n.IsNaN()) {
					// skip
				} else {
					tscType.Set("value", v)
				}
			}
		}

		// Handle enum literal
		if t.Flags()&checker.TypeFlagsEnumLiteral != 0 {
			// FIXME 'nameType' is just some random name from generated Kotlin TypeObjectProperty.
			// FIXME This field should have its own name.
			tscType.Set("nameType", getEnumQualifiedName(t))
		}

		// Handle template literal
		if t.Flags()&checker.TypeFlagsTemplateLiteral != 0 {
			templateLiteralType := t.AsTemplateLiteralType()
			tscType.Set("texts", templateLiteralType.Texts())
		}

		// Handle type parameter isThisType
		if t.Flags()&checker.TypeFlagsTypeParameter != 0 {
			typeParam := t.AsTypeParameter()
			if typeParam.IsThisType() {
				tscType.Set("isThisType", true)
			}
		}

		// Handle intrinsic name
		if t.Flags()&checker.TypeFlagsIntrinsic != 0 {
			intrinsicType := t.AsIntrinsicType()
			if intrinsicType != nil && intrinsicType.IntrinsicName() != "" {
				tscType.Set("intrinsicName", intrinsicType.IntrinsicName())
			}
		}

		// Handle tuple element flags
		if t.Flags()&checker.TypeFlagsObject != 0 && t.ObjectFlags()&checker.ObjectFlagsTuple != 0 {
			tupleType := t.AsTupleType()
			elementInfos := tupleType.ElementInfos()
			elementFlags := make([]interface{}, len(elementInfos))
			for i, elementInfo := range elementInfos {
				elementFlags[i] = convertElementFlags(elementInfo.ElementFlags())
			}
			tscType.Set("elementFlags", elementFlags)
			if combinedFlags := convertElementFlags(tupleType.CombinedFlags()); combinedFlags != 0 {
				tscType.Set("combinedFlags", combinedFlags)
			}
		}

		// Add type ID
		typeId := t.Id()
		tscType.Set("id", typeId)
		ctx.seenTypeIds[typeId] = t
		return tscType
	}, ctx)
}

func getSourceFileParent(node *ast.Node) *ast.Node {
	if ast.IsSourceFile(node) {
		return nil
	}
	current := node.Parent
	for current != nil {
		if ast.IsSourceFile(current) {
			return current
		}
		current = current.Parent
	}
	return nil
}

func ConvertNode(node *ast.Node, ctx *ConvertContext) *collections.OrderedMap[string, interface{}] {
	return FindReferenceOrConvert(node, func(ideObjectId int) *collections.OrderedMap[string, interface{}] {
		result := collections.NewOrderedMapWithSizeHint[string, interface{}](1)
		result.Set("ideObjectId", ideObjectId)

		if ast.IsSourceFile(node) {
			result.Set("ideObjectType", "SourceFileObject")
			result.Set("fileName", node.AsSourceFile().FileName())
			return result
		} else {
			result.Set("ideObjectType", "NodeObject")
			sourceFileParent := getSourceFileParent(node)
			if sourceFileParent == nil || node.Pos() == -1 || node.End() == -1 {
				if sourceFileParent != nil {
					result.Set("parent", ConvertNode(sourceFileParent, ctx))
				}
				return result
			}

			// Add range information
			if sourceFileParent != nil {
				sourceFile := sourceFileParent.AsSourceFile()
				startLine, startChar := scanner.GetECMALineAndCharacterOfPosition(sourceFile, node.Pos())
				endLine, endChar := scanner.GetECMALineAndCharacterOfPosition(sourceFile, node.End())
				result.Set("range", &lsproto.Range{
					Start: lsproto.Position{Line: uint32(startLine), Character: uint32(startChar)},
					End:   lsproto.Position{Line: uint32(endLine), Character: uint32(endChar)},
				})
			}

			// Add parent information
			if sourceFileParent != nil {
				result.Set("parent", ConvertNode(sourceFileParent, ctx))
			}

			// Check for computed property
			if node.Name() != nil && ast.IsComputedPropertyName(node.Name()) {
				result.Set("computedProperty", true)
			}
			return result
		}
	}, ctx)
}

func getEnumQualifiedName(t *checker.Type) string {
	if t == nil || t.Symbol() == nil {
		return ""
	}

	qName := ""
	current := t.Symbol().Parent
	for current != nil && !(current.ValueDeclaration != nil && ast.IsSourceFile(current.ValueDeclaration)) {
		if qName == "" {
			qName = current.Name
		} else {
			qName = current.Name + "." + qName
		}
		current = current.Parent
	}
	return qName
}

func ConvertPseudoBigInt(pseudoBigInt jsnum.PseudoBigInt, ctx *ConvertContext) *collections.OrderedMap[string, interface{}] {
	return FindReferenceOrConvert(pseudoBigInt, func(ideObjectId int) *collections.OrderedMap[string, interface{}] {
		result := collections.NewOrderedMapWithSizeHint[string, interface{}](4)
		result.Set("ideObjectId", ideObjectId)
		result.Set("negative", pseudoBigInt.Negative)
		result.Set("base10Value", pseudoBigInt.Base10Value)
		return result
	}, ctx)
}

func ConvertIndexInfo(indexInfo *checker.IndexInfo, ctx *ConvertContext) *collections.OrderedMap[string, interface{}] {
	return FindReferenceOrConvert(indexInfo, func(ideObjectId int) *collections.OrderedMap[string, interface{}] {
		result := collections.NewOrderedMapWithSizeHint[string, interface{}](7)
		result.Set("ideObjectId", ideObjectId)
		result.Set("ideObjectType", "IndexInfo")
		result.Set("keyType", ConvertType(indexInfo.KeyType(), ctx))
		result.Set("type", ConvertType(indexInfo.ValueType(), ctx))
		result.Set("isReadonly", indexInfo.IsReadonly())

		if indexInfo.Declaration() != nil {
			result.Set("declaration", ConvertNode(indexInfo.Declaration(), ctx))
		}
		return result
	}, ctx)
}

func ConvertSymbol(symbol *ast.Symbol, ctx *ConvertContext) *collections.OrderedMap[string, interface{}] {
	return FindReferenceOrConvert(symbol, func(ideObjectId int) *collections.OrderedMap[string, interface{}] {
		result := collections.NewOrderedMapWithSizeHint[string, interface{}](7)
		result.Set("ideObjectId", ideObjectId)
		result.Set("ideObjectType", "SymbolObject")
		result.Set("flags", convertSymbolFlags(symbol.Flags))
		escapedName := symbol.Name
		if strings.Contains(escapedName, ast.InternalSymbolNamePrefix) {
			escapedName = strings.ReplaceAll(escapedName, ast.InternalSymbolNamePrefix, "__")
		}
		result.Set("escapedName", escapedName)

		if symbol.Declarations != nil && len(symbol.Declarations) > 0 {
			declarations := make([]interface{}, 0)
			for _, d := range symbol.Declarations {
				declarations = append(declarations, ConvertNode(d, ctx))
			}
			result.Set("declarations", declarations)
		}

		if symbol.ValueDeclaration != nil {
			result.Set("valueDeclaration", ConvertNode(symbol.ValueDeclaration, ctx))
		}

		// TODO Handle symbol type
		// In TypeScript, this is checking for symbol.links?.type or symbol.type
		//symbolType := ctx.checker.GetTypeOfSymbolAtLocation(symbol, nil)
		//if symbolType != nil {
		//	result["type"] = ConvertType(symbolType, ctx)
		//}

		// Get symbol ID
		symbolId := ast.GetSymbolId(symbol)
		ctx.seenSymbolIds[symbolId] = symbol
		result.Set("id", symbolId)

		return result
	}, ctx)
}

func ConvertTypeProperties(t *checker.Type, ctx *ConvertContext) *collections.OrderedMap[string, interface{}] {
	return FindReferenceOrConvert(t, func(ideObjectId int) *collections.OrderedMap[string, interface{}] {
		prepared := collections.NewOrderedMapWithSizeHint[string, interface{}](10)
		prepared.Set("ideObjectId", ideObjectId)
		prepared.Set("ideObjectType", "TypeObject")
		prepared.Set("flags", convertTypeFlags(t.Flags()))
		prepared.Set("objectFlags", convertObjectFlags(t.ObjectFlags()))

		if t.Flags()&checker.TypeFlagsObject != 0 {
			assignObjectTypeProperties(t.AsObjectType(), ctx, prepared)
		}
		if t.Flags()&checker.TypeFlagsUnionOrIntersection != 0 {
			assignUnionOrIntersectionTypeProperties(t.AsUnionOrIntersectionType(), ctx, prepared)
		}
		if t.Flags()&checker.TypeFlagsConditional != 0 {
			assignConditionalTypeProperties(t.AsConditionalType(), ctx, prepared)
		}
		return prepared
	}, ctx)
}

func assignObjectTypeProperties(t *checker.ObjectType, ctx *ConvertContext, tscType *collections.OrderedMap[string, interface{}]) {
	constructSignatures := make([]interface{}, 0)
	for _, s := range ctx.checker.GetConstructSignatures(t.AsType()) {
		constructSignatures = append(constructSignatures, ConvertSignature(s, ctx))
	}
	tscType.Set("constructSignatures", constructSignatures)

	callSignatures := make([]interface{}, 0)
	for _, s := range ctx.checker.GetCallSignatures(t.AsType()) {
		callSignatures = append(callSignatures, ConvertSignature(s, ctx))
	}
	tscType.Set("callSignatures", callSignatures)

	properties := make([]interface{}, 0)
	for _, p := range ctx.checker.GetPropertiesOfType(t.AsType()) {
		properties = append(properties, ConvertSymbol(p, ctx))
	}
	tscType.Set("properties", properties)

	indexInfos := make([]interface{}, 0)
	for _, info := range ctx.checker.GetIndexInfosOfType(t.AsType()) {
		indexInfos = append(indexInfos, ConvertIndexInfo(info, ctx))
	}
	tscType.Set("indexInfos", indexInfos)
}

func assignUnionOrIntersectionTypeProperties(t *checker.UnionOrIntersectionType, ctx *ConvertContext, tscType *collections.OrderedMap[string, interface{}]) {
	resolvedProperties := make([]interface{}, 0)
	for _, p := range ctx.checker.GetPropertiesOfType(t.AsType()) {
		resolvedProperties = append(resolvedProperties, ConvertSymbol(p, ctx))
	}
	tscType.Set("resolvedProperties", resolvedProperties)

	callSignatures := make([]interface{}, 0)
	for _, s := range ctx.checker.GetCallSignatures(t.AsType()) {
		callSignatures = append(callSignatures, ConvertSignature(s, ctx))
	}
	tscType.Set("callSignatures", callSignatures)

	constructSignatures := make([]interface{}, 0)
	for _, s := range ctx.checker.GetConstructSignatures(t.AsType()) {
		constructSignatures = append(constructSignatures, ConvertSignature(s, ctx))
	}
	tscType.Set("constructSignatures", constructSignatures)
}

func assignConditionalTypeProperties(t *checker.ConditionalType, ctx *ConvertContext, tscType *collections.OrderedMap[string, interface{}]) {
	trueType := ctx.checker.GetTrueTypeFromConditionalType(t.AsType())
	if trueType != nil {
		tscType.Set("resolvedTrueType", ConvertType(trueType, ctx))
	}

	falseType := ctx.checker.GetFalseTypeFromConditionalType(t.AsType())
	if falseType != nil {
		tscType.Set("resolvedFalseType", ConvertType(falseType, ctx))
	}
}

func ConvertSignature(signature *checker.Signature, ctx *ConvertContext) *collections.OrderedMap[string, interface{}] {
	return FindReferenceOrConvert(signature, func(ideObjectId int) *collections.OrderedMap[string, interface{}] {
		result := collections.NewOrderedMapWithSizeHint[string, interface{}](5)
		result.Set("ideObjectId", ideObjectId)
		result.Set("ideObjectType", "SignatureObject")

		if declaration := signature.Declaration(); declaration != nil {
			result.Set("declaration", ConvertNode(declaration, ctx))
		}

		if returnType := ctx.checker.GetReturnTypeOfSignature(signature); returnType != nil {
			result.Set("resolvedReturnType", ConvertType(returnType, ctx))
		}

		parameters := make([]interface{}, 0)
		for _, param := range signature.Parameters() {
			parameters = append(parameters, ConvertSymbol(param, ctx))
		}
		result.Set("parameters", parameters)

		result.Set("flags", convertSignatureFlags(signature))

		return result
	}, ctx)
}

func typeRequestKindToContextFlags(typeRequestKind lsproto.TypeRequestKind) int {
	switch typeRequestKind {
	case lsproto.TypeRequestKindDefault:
		return -1
	case lsproto.TypeRequestKindContextual:
		return 0
	case lsproto.TypeRequestKindContextualCompletions:
		return 4
	default:
		panic(fmt.Sprintf("Unexpected typeRequestKind %s", typeRequestKind))
	}
}
