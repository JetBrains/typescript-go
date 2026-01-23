package ls

import (
	"context"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/ls/lsutil"
)

func (ls *LanguageService) GetCompletionDataSymbols(
	ctx context.Context,
	typeChecker *checker.Checker,
	file *ast.SourceFile,
	position int,
	preferences *lsutil.UserPreferences,
) []*ast.Symbol {
	data, _ := ls.getCompletionData(ctx, typeChecker, file, position, preferences)
	switch d := data.(type) {
	case *completionDataData:
		return d.symbols
	default:
		return nil
	}
}
