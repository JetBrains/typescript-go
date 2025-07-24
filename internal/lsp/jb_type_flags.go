package lsp

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
)

type TsTypeFlags uint32

const (
	TsTypeFlagsAny             TsTypeFlags = 1 << 0
	TsTypeFlagsUnknown         TsTypeFlags = 1 << 1
	TsTypeFlagsString          TsTypeFlags = 1 << 2
	TsTypeFlagsNumber          TsTypeFlags = 1 << 3
	TsTypeFlagsBoolean         TsTypeFlags = 1 << 4
	TsTypeFlagsEnum            TsTypeFlags = 1 << 5 // Numeric computed enum member value
	TsTypeFlagsBigInt          TsTypeFlags = 1 << 6
	TsTypeFlagsStringLiteral   TsTypeFlags = 1 << 7
	TsTypeFlagsNumberLiteral   TsTypeFlags = 1 << 8
	TsTypeFlagsBooleanLiteral  TsTypeFlags = 1 << 9
	TsTypeFlagsEnumLiteral     TsTypeFlags = 1 << 10 // Always combined with StringLiteral, NumberLiteral, or Union
	TsTypeFlagsBigIntLiteral   TsTypeFlags = 1 << 11
	TsTypeFlagsESSymbol        TsTypeFlags = 1 << 12 // Type of symbol primitive introduced in ES6
	TsTypeFlagsUniqueESSymbol  TsTypeFlags = 1 << 13 // unique symbol
	TsTypeFlagsVoid            TsTypeFlags = 1 << 14
	TsTypeFlagsUndefined       TsTypeFlags = 1 << 15
	TsTypeFlagsNull            TsTypeFlags = 1 << 16
	TsTypeFlagsNever           TsTypeFlags = 1 << 17 // Never type
	TsTypeFlagsTypeParameter   TsTypeFlags = 1 << 18 // Type parameter
	TsTypeFlagsObject          TsTypeFlags = 1 << 19 // Object type
	TsTypeFlagsUnion           TsTypeFlags = 1 << 20 // Union (T | U)
	TsTypeFlagsIntersection    TsTypeFlags = 1 << 21 // Intersection (T & U)
	TsTypeFlagsIndex           TsTypeFlags = 1 << 22 // keyof T
	TsTypeFlagsIndexedAccess   TsTypeFlags = 1 << 23 // T[K]
	TsTypeFlagsConditional     TsTypeFlags = 1 << 24 // T extends U ? X : Y
	TsTypeFlagsSubstitution    TsTypeFlags = 1 << 25 // Type parameter substitution
	TsTypeFlagsNonPrimitive    TsTypeFlags = 1 << 26 // intrinsic object type
	TsTypeFlagsTemplateLiteral TsTypeFlags = 1 << 27 // Template literal type
	TsTypeFlagsStringMapping   TsTypeFlags = 1 << 28 // Uppercase/Lowercase type
)

func convertTypeFlags(goFlags checker.TypeFlags) TsTypeFlags {
	var result TsTypeFlags = 0
	if goFlags&checker.TypeFlagsAny != 0 {
		result = result | TsTypeFlagsAny
	}
	if goFlags&checker.TypeFlagsUnknown != 0 {
		result = result | TsTypeFlagsUnknown
	}
	if goFlags&checker.TypeFlagsUndefined != 0 {
		result = result | TsTypeFlagsUndefined
	}
	if goFlags&checker.TypeFlagsNull != 0 {
		result = result | TsTypeFlagsNull
	}
	if goFlags&checker.TypeFlagsVoid != 0 {
		result = result | TsTypeFlagsVoid
	}
	if goFlags&checker.TypeFlagsString != 0 {
		result = result | TsTypeFlagsString
	}
	if goFlags&checker.TypeFlagsNumber != 0 {
		result = result | TsTypeFlagsNumber
	}
	if goFlags&checker.TypeFlagsBigInt != 0 {
		result = result | TsTypeFlagsBigInt
	}
	if goFlags&checker.TypeFlagsBoolean != 0 {
		result = result | TsTypeFlagsBoolean
	}
	if goFlags&checker.TypeFlagsESSymbol != 0 {
		result = result | TsTypeFlagsESSymbol
	}
	if goFlags&checker.TypeFlagsStringLiteral != 0 {
		result = result | TsTypeFlagsStringLiteral
	}
	if goFlags&checker.TypeFlagsNumberLiteral != 0 {
		result = result | TsTypeFlagsNumberLiteral
	}
	if goFlags&checker.TypeFlagsBigIntLiteral != 0 {
		result = result | TsTypeFlagsBigIntLiteral
	}
	if goFlags&checker.TypeFlagsBooleanLiteral != 0 {
		result = result | TsTypeFlagsBooleanLiteral
	}
	if goFlags&checker.TypeFlagsUniqueESSymbol != 0 {
		result = result | TsTypeFlagsUniqueESSymbol
	}
	if goFlags&checker.TypeFlagsEnumLiteral != 0 {
		result = result | TsTypeFlagsEnumLiteral
	}
	if goFlags&checker.TypeFlagsEnum != 0 {
		result = result | TsTypeFlagsEnum
	}
	if goFlags&checker.TypeFlagsNever != 0 {
		result = result | TsTypeFlagsNever
	}
	if goFlags&checker.TypeFlagsTypeParameter != 0 {
		result = result | TsTypeFlagsTypeParameter
	}
	if goFlags&checker.TypeFlagsObject != 0 {
		result = result | TsTypeFlagsObject
	}
	if goFlags&checker.TypeFlagsUnion != 0 {
		result = result | TsTypeFlagsUnion
	}
	if goFlags&checker.TypeFlagsIntersection != 0 {
		result = result | TsTypeFlagsIntersection
	}
	if goFlags&checker.TypeFlagsIndex != 0 {
		result = result | TsTypeFlagsIndex
	}
	if goFlags&checker.TypeFlagsIndexedAccess != 0 {
		result = result | TsTypeFlagsIndexedAccess
	}
	if goFlags&checker.TypeFlagsConditional != 0 {
		result = result | TsTypeFlagsConditional
	}
	if goFlags&checker.TypeFlagsSubstitution != 0 {
		result = result | TsTypeFlagsSubstitution
	}
	if goFlags&checker.TypeFlagsNonPrimitive != 0 {
		result = result | TsTypeFlagsNonPrimitive
	}
	if goFlags&checker.TypeFlagsTemplateLiteral != 0 {
		result = result | TsTypeFlagsTemplateLiteral
	}
	if goFlags&checker.TypeFlagsStringMapping != 0 {
		result = result | TsTypeFlagsStringMapping
	}
	return result
}

type TsSymbolFlags uint32

const (
	TsSymbolFlagsFunctionScopedVariable TsSymbolFlags = 1 << 0  // Variable (var) or parameter
	TsSymbolFlagsBlockScopedVariable    TsSymbolFlags = 1 << 1  // A block-scoped variable (let or const)
	TsSymbolFlagsProperty               TsSymbolFlags = 1 << 2  // Property or enum member
	TsSymbolFlagsEnumMember             TsSymbolFlags = 1 << 3  // Enum member
	TsSymbolFlagsFunction               TsSymbolFlags = 1 << 4  // Function
	TsSymbolFlagsClass                  TsSymbolFlags = 1 << 5  // Class
	TsSymbolFlagsInterface              TsSymbolFlags = 1 << 6  // Interface
	TsSymbolFlagsConstEnum              TsSymbolFlags = 1 << 7  // Const enum
	TsSymbolFlagsRegularEnum            TsSymbolFlags = 1 << 8  // Enum
	TsSymbolFlagsValueModule            TsSymbolFlags = 1 << 9  // Instantiated module
	TsSymbolFlagsNamespaceModule        TsSymbolFlags = 1 << 10 // Uninstantiated module
	TsSymbolFlagsTypeLiteral            TsSymbolFlags = 1 << 11 // Type Literal or mapped type
	TsSymbolFlagsObjectLiteral          TsSymbolFlags = 1 << 12 // Object Literal
	TsSymbolFlagsMethod                 TsSymbolFlags = 1 << 13 // Method
	TsSymbolFlagsConstructor            TsSymbolFlags = 1 << 14 // Constructor
	TsSymbolFlagsGetAccessor            TsSymbolFlags = 1 << 15 // Get accessor
	TsSymbolFlagsSetAccessor            TsSymbolFlags = 1 << 16 // Set accessor
	TsSymbolFlagsSignature              TsSymbolFlags = 1 << 17 // Call, construct, or index signature
	TsSymbolFlagsTypeParameter          TsSymbolFlags = 1 << 18 // Type parameter
	TsSymbolFlagsTypeAlias              TsSymbolFlags = 1 << 19 // Type alias
	TsSymbolFlagsExportValue            TsSymbolFlags = 1 << 20 // Exported value marker (see comment in declareModuleMember in binder)
	TsSymbolFlagsAlias                  TsSymbolFlags = 1 << 21 // An alias for another symbol (see comment in isAliasSymbolDeclaration in checker)
	TsSymbolFlagsPrototype              TsSymbolFlags = 1 << 22 // Prototype property (no source representation)
	TsSymbolFlagsExportStar             TsSymbolFlags = 1 << 23 // Export * declaration
	TsSymbolFlagsOptional               TsSymbolFlags = 1 << 24 // Optional property
	TsSymbolFlagsTransient              TsSymbolFlags = 1 << 25 // Transient symbol (created during type check)
	TsSymbolFlagsAssignment             TsSymbolFlags = 1 << 26 // Assignment treated as declaration (eg `this.prop = 1`)
	TsSymbolFlagsModuleExports          TsSymbolFlags = 1 << 27 // Symbol for CommonJS `module` of `module.exports`
)

func convertSymbolFlags(goFlags ast.SymbolFlags) TsSymbolFlags {
	var result TsSymbolFlags = 0
	if goFlags&ast.SymbolFlagsFunctionScopedVariable != 0 {
		result = result | TsSymbolFlagsFunctionScopedVariable
	}
	if goFlags&ast.SymbolFlagsBlockScopedVariable != 0 {
		result = result | TsSymbolFlagsBlockScopedVariable
	}
	if goFlags&ast.SymbolFlagsProperty != 0 {
		result = result | TsSymbolFlagsProperty
	}
	if goFlags&ast.SymbolFlagsEnumMember != 0 {
		result = result | TsSymbolFlagsEnumMember
	}
	if goFlags&ast.SymbolFlagsFunction != 0 {
		result = result | TsSymbolFlagsFunction
	}
	if goFlags&ast.SymbolFlagsClass != 0 {
		result = result | TsSymbolFlagsClass
	}
	if goFlags&ast.SymbolFlagsInterface != 0 {
		result = result | TsSymbolFlagsInterface
	}
	if goFlags&ast.SymbolFlagsConstEnum != 0 {
		result = result | TsSymbolFlagsConstEnum
	}
	if goFlags&ast.SymbolFlagsRegularEnum != 0 {
		result = result | TsSymbolFlagsRegularEnum
	}
	if goFlags&ast.SymbolFlagsValueModule != 0 {
		result = result | TsSymbolFlagsValueModule
	}
	if goFlags&ast.SymbolFlagsNamespaceModule != 0 {
		result = result | TsSymbolFlagsNamespaceModule
	}
	if goFlags&ast.SymbolFlagsTypeLiteral != 0 {
		result = result | TsSymbolFlagsTypeLiteral
	}
	if goFlags&ast.SymbolFlagsObjectLiteral != 0 {
		result = result | TsSymbolFlagsObjectLiteral
	}
	if goFlags&ast.SymbolFlagsMethod != 0 {
		result = result | TsSymbolFlagsMethod
	}
	if goFlags&ast.SymbolFlagsConstructor != 0 {
		result = result | TsSymbolFlagsConstructor
	}
	if goFlags&ast.SymbolFlagsGetAccessor != 0 {
		result = result | TsSymbolFlagsGetAccessor
	}
	if goFlags&ast.SymbolFlagsSetAccessor != 0 {
		result = result | TsSymbolFlagsSetAccessor
	}
	if goFlags&ast.SymbolFlagsSignature != 0 {
		result = result | TsSymbolFlagsSignature
	}
	if goFlags&ast.SymbolFlagsTypeParameter != 0 {
		result = result | TsSymbolFlagsTypeParameter
	}
	if goFlags&ast.SymbolFlagsTypeAlias != 0 {
		result = result | TsSymbolFlagsTypeAlias
	}
	if goFlags&ast.SymbolFlagsExportValue != 0 {
		result = result | TsSymbolFlagsExportValue
	}
	if goFlags&ast.SymbolFlagsAlias != 0 {
		result = result | TsSymbolFlagsAlias
	}
	if goFlags&ast.SymbolFlagsPrototype != 0 {
		result = result | TsSymbolFlagsPrototype
	}
	if goFlags&ast.SymbolFlagsExportStar != 0 {
		result = result | TsSymbolFlagsExportStar
	}
	if goFlags&ast.SymbolFlagsOptional != 0 {
		result = result | TsSymbolFlagsOptional
	}
	if goFlags&ast.SymbolFlagsTransient != 0 {
		result = result | TsSymbolFlagsTransient
	}
	if goFlags&ast.SymbolFlagsAssignment != 0 {
		result = result | TsSymbolFlagsAssignment
	}
	if goFlags&ast.SymbolFlagsModuleExports != 0 {
		result = result | TsSymbolFlagsModuleExports
	}
	// if goFlags&ast.SymbolFlagsConstEnumOnlyModule != 0 {
	// 	result = result | TsSymbolFlagsConstEnumOnlyModule
	// }
	// if goFlags&ast.SymbolFlagsReplaceableByMethod != 0 {
	// 	result = result | TsSymbolFlagsReplaceableByMethod
	// }
	// if goFlags&ast.SymbolFlagsGlobalLookup != 0 {
	// 	result = result | TsSymbolFlagsGlobalLookup
	// }

	return result
}

type TsObjectFlags uint32

const (
	TsObjectFlagsClass                                      TsObjectFlags = 1 << 0  // Class
	TsObjectFlagsInterface                                  TsObjectFlags = 1 << 1  // Interface
	TsObjectFlagsReference                                  TsObjectFlags = 1 << 2  // Generic type reference
	TsObjectFlagsTuple                                      TsObjectFlags = 1 << 3  // Synthesized generic tuple type
	TsObjectFlagsAnonymous                                  TsObjectFlags = 1 << 4  // Anonymous
	TsObjectFlagsMapped                                     TsObjectFlags = 1 << 5  // Mapped
	TsObjectFlagsInstantiated                               TsObjectFlags = 1 << 6  // Instantiated anonymous or mapped type
	TsObjectFlagsObjectLiteral                              TsObjectFlags = 1 << 7  // Originates in an object literal
	TsObjectFlagsEvolvingArray                              TsObjectFlags = 1 << 8  // Evolving array type
	TsObjectFlagsObjectLiteralPatternWithComputedProperties TsObjectFlags = 1 << 9  // Object literal pattern with computed properties
	TsObjectFlagsReverseMapped                              TsObjectFlags = 1 << 10 // Object contains a property from a reverse-mapped type
	TsObjectFlagsJsxAttributes                              TsObjectFlags = 1 << 11 // Jsx attributes type
	TsObjectFlagsJSLiteral                                  TsObjectFlags = 1 << 12 // Object type declared in JS - disables errors on read/write of nonexisting members
	TsObjectFlagsFreshLiteral                               TsObjectFlags = 1 << 13 // Fresh object literal
	TsObjectFlagsArrayLiteral                               TsObjectFlags = 1 << 14 // Originates in an array literal
	TsObjectFlagsPrimitiveUnion                             TsObjectFlags = 1 << 15 // Union of only primitive types
	TsObjectFlagsContainsWideningType                       TsObjectFlags = 1 << 16 // Type is or contains undefined or null widening type
	TsObjectFlagsContainsObjectOrArrayLiteral               TsObjectFlags = 1 << 17 // Type is or contains object literal type
	TsObjectFlagsNonInferrableType                          TsObjectFlags = 1 << 18 // Type is or contains anyFunctionType or silentNeverType
	TsObjectFlagsCouldContainTypeVariablesComputed          TsObjectFlags = 1 << 19 // CouldContainTypeVariables flag has been computed
	TsObjectFlagsCouldContainTypeVariables                  TsObjectFlags = 1 << 20 // Type could contain a type variable
)

func convertObjectFlags(goFlags checker.ObjectFlags) TsObjectFlags {
	var result TsObjectFlags = 0
	if goFlags&checker.ObjectFlagsClass != 0 {
		result = result | TsObjectFlagsClass
	}
	if goFlags&checker.ObjectFlagsInterface != 0 {
		result = result | TsObjectFlagsInterface
	}
	if goFlags&checker.ObjectFlagsReference != 0 {
		result = result | TsObjectFlagsReference
	}
	if goFlags&checker.ObjectFlagsTuple != 0 {
		result = result | TsObjectFlagsTuple
	}
	if goFlags&checker.ObjectFlagsAnonymous != 0 {
		result = result | TsObjectFlagsAnonymous
	}
	if goFlags&checker.ObjectFlagsMapped != 0 {
		result = result | TsObjectFlagsMapped
	}
	if goFlags&checker.ObjectFlagsInstantiated != 0 {
		result = result | TsObjectFlagsInstantiated
	}
	if goFlags&checker.ObjectFlagsObjectLiteral != 0 {
		result = result | TsObjectFlagsObjectLiteral
	}
	if goFlags&checker.ObjectFlagsEvolvingArray != 0 {
		result = result | TsObjectFlagsEvolvingArray
	}
	if goFlags&checker.ObjectFlagsObjectLiteralPatternWithComputedProperties != 0 {
		result = result | TsObjectFlagsObjectLiteralPatternWithComputedProperties
	}
	if goFlags&checker.ObjectFlagsReverseMapped != 0 {
		result = result | TsObjectFlagsReverseMapped
	}
	if goFlags&checker.ObjectFlagsJsxAttributes != 0 {
		result = result | TsObjectFlagsJsxAttributes
	}
	if goFlags&checker.ObjectFlagsJSLiteral != 0 {
		result = result | TsObjectFlagsJSLiteral
	}
	if goFlags&checker.ObjectFlagsFreshLiteral != 0 {
		result = result | TsObjectFlagsFreshLiteral
	}
	if goFlags&checker.ObjectFlagsArrayLiteral != 0 {
		result = result | TsObjectFlagsArrayLiteral
	}
	if goFlags&checker.ObjectFlagsPrimitiveUnion != 0 {
		result = result | TsObjectFlagsPrimitiveUnion
	}
	if goFlags&checker.ObjectFlagsContainsWideningType != 0 {
		result = result | TsObjectFlagsContainsWideningType
	}
	if goFlags&checker.ObjectFlagsContainsObjectOrArrayLiteral != 0 {
		result = result | TsObjectFlagsContainsObjectOrArrayLiteral
	}
	if goFlags&checker.ObjectFlagsNonInferrableType != 0 {
		result = result | TsObjectFlagsNonInferrableType
	}
	if goFlags&checker.ObjectFlagsCouldContainTypeVariablesComputed != 0 {
		result = result | TsObjectFlagsCouldContainTypeVariablesComputed
	}
	if goFlags&checker.ObjectFlagsCouldContainTypeVariables != 0 {
		result = result | TsObjectFlagsCouldContainTypeVariables
	}
	//if goFlags&checker.ObjectFlagsMembersResolved != 0 {
	//	result = result | TsObjectFlagsMembersResolved
	//}
	return result
}

type TsElementFlags uint32

const (
	TsElementFlagsRequired TsElementFlags = 1 << 0 // T
	TsElementFlagsOptional TsElementFlags = 1 << 1 // T?
	TsElementFlagsRest     TsElementFlags = 1 << 2 // ...T[]
	TsElementFlagsVariadic TsElementFlags = 1 << 3 // ...T
)

func convertElementFlags(goFlags checker.ElementFlags) TsElementFlags {
	var result TsElementFlags = 0
	if goFlags&checker.ElementFlagsRequired != 0 {
		result = result | TsElementFlagsRequired
	}
	if goFlags&checker.ElementFlagsOptional != 0 {
		result = result | TsElementFlagsOptional
	}
	if goFlags&checker.ElementFlagsRest != 0 {
		result = result | TsElementFlagsRest
	}
	if goFlags&checker.ElementFlagsVariadic != 0 {
		result = result | TsElementFlagsVariadic
	}
	return result
}

type TsSignatureFlags uint32

const (
	TsIsSignatureCandidateForOverloadFailure TsSignatureFlags = 1 << 7
)

func convertSignatureFlags(signature *checker.Signature) TsSignatureFlags {
	var result TsSignatureFlags = 0
	if signature.IsSignatureCandidateForOverloadFailure() {
		result = result | TsIsSignatureCandidateForOverloadFailure
	}
	return result
}
