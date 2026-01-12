// MODIFIED IN THIS FORK: 2025-09-12 Konstantin Ulitin e3893948a2061ee66b3bc9e3df22590aba344872 support ideGetTypeProperty command
// MODIFIED IN THIS FORK: 2025-07-24 Aleksei Berezkin 5567c01219bf93fad344a80691da5df021923d10 WEB-74070 Supported the method IdeGetResolvedSignature
// MODIFIED IN THIS FORK: 2025-05-21 Piotr Tomiak dfd337df3e64d6fb2bc18c46bf4de05bcbea5a64 Support WebStorm types in LSP mode
package lsproto

import (
	//nolint
	"encoding/json"
	"fmt"
)

const (
	MethodJbHandleCustomTsServerCommand Method = "$/ij/handleCustomTsServerCommand"
)

type JbHandleCustomTsServerCommandParams struct {
	IdeCommand IdeCommand  `json:"ideCommand"`
	Arguments  interface{} `json:"args"`
}

type IdeCommand string

const (
	IdeCommandGetElementType       IdeCommand = "ideGetElementType"
	IdeCommandGetSymbolType        IdeCommand = "ideGetSymbolType"
	IdeCommandGetTypeProperties    IdeCommand = "ideGetTypeProperties"
	IdeCommandGetTypeProperty      IdeCommand = "ideGetTypeProperty"
	IdeCommandGetTypeText          IdeCommand = "ideGetTypeText"
	IdeAreTypesMutuallyAssignable  IdeCommand = "ideAreTypesMutuallyAssignable"
	IdeGetResolvedSignature        IdeCommand = "ideGetResolvedSignature"
	IdeCommandGetCompletionSymbols IdeCommand = "ideGetCompletionSymbols"
)

type TypeRequestKind string

const (
	TypeRequestKindDefault               TypeRequestKind = "Default"
	TypeRequestKindContextual            TypeRequestKind = "Contextual"
	TypeRequestKindContextualCompletions TypeRequestKind = "ContextualCompletions"
)

type GetElementTypeArguments struct {
	File            DocumentUri     `json:"file"`
	Range           Range           `json:"range"`
	TypeRequestKind TypeRequestKind `json:"typeRequestKind"`
	ProjectFileName *DocumentUri    `json:"projectFileName,omitempty"`
	ForceReturnType bool            `json:"forceReturnType"`
}

type GetSymbolTypeArguments struct {
	IdeTypeCheckerId int `json:"ideTypeCheckerId"`
	IdeProjectId     int `json:"ideProjectId"`
	SymbolId         int `json:"symbolId"`
}

type GetTypePropertiesArguments struct {
	IdeTypeCheckerId int `json:"ideTypeCheckerId"`
	IdeProjectId     int `json:"ideProjectId"`
	TypeId           int `json:"typeId"`
}

type GetTypePropertyArguments struct {
	IdeTypeCheckerId int    `json:"ideTypeCheckerId"`
	IdeProjectId     int    `json:"ideProjectId"`
	TypeId           int    `json:"typeId"`
	PropertyName     string `json:"propertyName"`
}

type GetTypeTextArguments struct {
	IdeTypeCheckerId int  `json:"ideTypeCheckerId"`
	IdeProjectId     int  `json:"ideProjectId"`
	SymbolId         int  `json:"symbolId"`
	Flags            *int `json:"flags,omitempty"`
}

type AreTypesMutuallyAssignableArguments struct {
	IdeTypeCheckerId int `json:"ideTypeCheckerId"`
	IdeProjectId     int `json:"ideProjectId"`
	Type1Id          int `json:"type1Id"`
	Type2Id          int `json:"type2Id"`
}

type GetResolvedSignatureArguments struct {
	File            DocumentUri  `json:"file"`
	Range           Range        `json:"range"`
	ProjectFileName *DocumentUri `json:"projectFileName"`
}

type GetCompletionSymbolsArguments struct {
	File            DocumentUri  `json:"file"`
	Position        Position     `json:"position"`
	ProjectFileName *DocumentUri `json:"projectFileName"`
}

func (p *JbHandleCustomTsServerCommandParams) UnmarshalJSON(data []byte) error {
	// First unmarshal into a temporary structure to get the ideCommand
	type TempParams struct {
		IdeCommand IdeCommand      `json:"ideCommand"`
		Arguments  json.RawMessage `json:"args"`
	}

	var temp TempParams
	if err := json.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("failed to unmarshal JbHandleCustomTsServerCommandParams: %w", err)
	}

	// Set the ideCommand
	p.IdeCommand = temp.IdeCommand

	// Based on ideCommand, unmarshal args into the appropriate type
	var args interface{}
	switch temp.IdeCommand {
	case IdeCommandGetElementType:
		var typedArgs GetElementTypeArguments
		if err := json.Unmarshal(temp.Arguments, &typedArgs); err != nil {
			return fmt.Errorf("failed to unmarshal GetElementTypeArguments: %w", err)
		}
		args = &typedArgs

	case IdeCommandGetSymbolType:
		var typedArgs GetSymbolTypeArguments
		if err := json.Unmarshal(temp.Arguments, &typedArgs); err != nil {
			return fmt.Errorf("failed to unmarshal GetSymbolTypeArguments: %w", err)
		}
		args = &typedArgs

	case IdeCommandGetTypeProperties:
		var typedArgs GetTypePropertiesArguments
		if err := json.Unmarshal(temp.Arguments, &typedArgs); err != nil {
			return fmt.Errorf("failed to unmarshal GetTypePropertiesArguments: %w", err)
		}
		args = &typedArgs

	case IdeCommandGetTypeProperty:
		var typedArgs GetTypePropertyArguments
		if err := json.Unmarshal(temp.Arguments, &typedArgs); err != nil {
			return fmt.Errorf("failed to unmarshal GetTypePropertyArguments: %w", err)
		}
		args = &typedArgs

	case IdeCommandGetTypeText:
		var typedArgs GetTypeTextArguments
		if err := json.Unmarshal(temp.Arguments, &typedArgs); err != nil {
			return fmt.Errorf("failed to unmarshal GetTypeTextArguments: %w", err)
		}
		args = &typedArgs

	case IdeAreTypesMutuallyAssignable:
		var typedArgs AreTypesMutuallyAssignableArguments
		if err := json.Unmarshal(temp.Arguments, &typedArgs); err != nil {
			return fmt.Errorf("failed to unmarshal AreTypesMutuallyAssignableArguments: %w", err)
		}
		args = &typedArgs

	case IdeGetResolvedSignature:
		var typedArgs GetResolvedSignatureArguments
		if err := json.Unmarshal(temp.Arguments, &typedArgs); err != nil {
			return fmt.Errorf("failed to unmarshal GetResolvedSignatureArguments: %w", err)
		}
		args = &typedArgs

	case IdeCommandGetCompletionSymbols:
		var typedArgs GetCompletionSymbolsArguments
		if err := json.Unmarshal(temp.Arguments, &typedArgs); err != nil {
			return fmt.Errorf("failed to unmarshal GetCompletionSymbolsArguments: %w", err)
		}
		args = &typedArgs

	default:
		return fmt.Errorf("unknown ideCommand: %s", temp.IdeCommand)
	}

	p.Arguments = args
	return nil
}
