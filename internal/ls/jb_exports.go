package ls

import (
	"context"
	"strings"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/ls/lsutil"
	"github.com/microsoft/typescript-go/internal/lsp/lsproto"
)

func (ls *LanguageService) GetCompletionDataSymbols(
	ctx context.Context,
	typeChecker *checker.Checker,
	file *ast.SourceFile,
	position int,
	preferences *lsutil.UserPreferences,
) []*ast.Symbol {
	data, _ := ls.getCompletionData(ctx, typeChecker, file, position, preferences, false)
	switch d := data.(type) {
	case *completionDataData:
		return d.symbols
	default:
		return nil
	}
}

type CompletionInfoWithSymbols struct {
	Items   []*lsproto.CompletionItem
	Symbols []*ast.Symbol
}

func (ls *LanguageService) GetCompletionInfoWithSymbols(
	ctx context.Context,
	checker *checker.Checker,
	file *ast.SourceFile,
	position int,
	preferences *lsutil.UserPreferences,
) *CompletionInfoWithSymbols {
	data, _ := ls.getCompletionData(ctx, checker, file, position, preferences, false)
	switch data := data.(type) {
	case *completionDataData:
		optionalReplacementSpan := ls.getOptionalReplacementSpan(data.location, file)
		response, err := ls.completionInfoFromData(
			ctx,
			checker,
			file,
			ls.GetProgram().Options(),
			data,
			position,
			optionalReplacementSpan,
		)
		if err != nil {
			return nil
		}
		return &CompletionInfoWithSymbols{
			Items:   response.Items,
			Symbols: matchSymbolsToItems(response.Items, data.symbols),
		}
	default:
		return nil
	}
}

// itemSymbolName returns the original symbol name from a CompletionItem.
// CompletionItem.Label may have a trailing '?' for optional properties
// (added in createLSPCompletionItem), so strip the '?' suffix.
func itemSymbolName(item *lsproto.CompletionItem) string {
	return strings.TrimSuffix(item.Label, "?")
}

// matchSymbolsToItems filters symbols to match items by index.
// Both slices are ordered consistently, but symbols may contain extra entries
// that were filtered out during item creation. This uses two pointers to align
// them by name. When a name appears a different number of times in items vs
// symbols, those symbols are excluded since we can't reliably match them.
func matchSymbolsToItems(items []*lsproto.CompletionItem, symbols []*ast.Symbol) []*ast.Symbol {
	// Count name occurrences in items
	itemNameCounts := make(map[string]int)
	for _, item := range items {
		itemNameCounts[itemSymbolName(item)]++
	}

	// Count name occurrences in symbols
	symbolNameCounts := make(map[string]int)
	for _, sym := range symbols {
		symbolNameCounts[ast.SymbolName(sym)]++
	}

	// Find names where counts match — only these can be reliably paired
	matchableNames := make(map[string]bool)
	for name, itemCount := range itemNameCounts {
		if symCount, ok := symbolNameCounts[name]; ok && symCount == itemCount {
			matchableNames[name] = true
		}
	}

	// Two-pointer walk: for each item, find the corresponding symbol
	result := make([]*ast.Symbol, 0, len(items))
	si := 0
	for _, item := range items {
		name := itemSymbolName(item)
		if !matchableNames[name] {
			continue
		}
		// Advance symbol pointer to find matching symbol
		for si < len(symbols) {
			symName := ast.SymbolName(symbols[si])
			if symName == name {
				result = append(result, symbols[si])
				si++
				break
			}
			si++
		}
	}

	return result
}
