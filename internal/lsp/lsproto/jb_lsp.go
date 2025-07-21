package lsproto

import (
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
	IdeCommandGetElementType      IdeCommand = "ideGetElementType"
	IdeCommandGetSymbolType       IdeCommand = "ideGetSymbolType"
	IdeCommandGetTypeProperties   IdeCommand = "ideGetTypeProperties"
	IdeAreTypesMutuallyAssignable IdeCommand = "ideAreTypesMutuallyAssignable"
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

type AreTypesMutuallyAssignableArguments struct {
	IdeTypeCheckerId int `json:"ideTypeCheckerId"`
	IdeProjectId     int `json:"ideProjectId"`
	Type1Id          int `json:"type1Id"`
	Type2Id          int `json:"type2Id"`
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

	case IdeAreTypesMutuallyAssignable:
		var typedArgs AreTypesMutuallyAssignableArguments
		if err := json.Unmarshal(temp.Arguments, &typedArgs); err != nil {
			return fmt.Errorf("failed to unmarshal AreTypesMutuallyAssignableArguments: %w", err)
		}
		args = &typedArgs

	default:
		return fmt.Errorf("unknown ideCommand: %s", temp.IdeCommand)
	}

	p.Arguments = args
	return nil
}
