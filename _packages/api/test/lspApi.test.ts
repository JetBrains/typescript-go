import assert from "node:assert"
import { Buffer } from "node:buffer"
import fs from "node:fs"
import subprocess from "node:child_process"
import path from "node:path"
import { afterEach, beforeEach, suite, test } from "node:test"
import url from "node:url"
import type { Range } from 'vscode-languageserver-types'
import type { Type, Symbol, Node, TypeReference, GenericType, UnionType, LiteralType, IndexType, IndexedAccessType, ConditionalType, SubstitutionType, ObjectType, PseudoBigInt, BigIntLiteralType, TemplateLiteral, TemplateLiteralType, TupleType, Signature, IndexInfo } from 'typescript'
// @ts-expect-error
import { TypeFlags } from '../src/typeFlags.ts'

suite("TypeScriptGoServiceGetElementTypeTest", () => {
  let client: LspClient | undefined

  beforeEach(async () => {
    try {
      client = new LspClient()
      await client.spawnAndInit()
    } catch (e) {
      await client?.kill()
      client = undefined
      throw e
    }
  })

  afterEach(async () => {
    try {
      await client?.shutdown()
    } catch (e) {
      await client?.kill()
      throw e
    } finally {
      client = undefined
    }
  })

  async function doTestElementType(content: string, elementText: string) {
    client!.didOpen("a.ts", content)
    return await client!.getElementType("a.ts", getRange(content, elementText))
  }


  test("testNumber", async () => {
    const type = await doTestElementType("declare const foo: number", "foo")
    // 8 is a converted flag, use direct instead
    assert.strictEqual(type.flags, 8)
  })

  test("testNumberLiteral", async () => {
    const type = await doTestElementType("const foo = 123", "foo")
    // 256 is a converted flag
    assert.strictEqual(type.flags, 256)
    assert.strictEqual((type as LiteralType).value, 123)
  })

  test("testObjectOptionalProperty", async () => {
    const type = await doTestElementType("type Foo = {x?: 1}", "Foo")
    assert.strictEqual(type.flags, TypeFlags.Object)
    assert.strictEqual((type as ObjectType).objectFlags, 16);
    assert.deepStrictEqual(await (type as TypeEx).getPropertiesEx(), [{}]) // wip mapping
    assert.deepStrictEqual(await (type as TypeEx).getCallSignaturesEx(), [])
    assert.deepStrictEqual(await (type as TypeEx).getConstructSignaturesEx(), [])
  })
})

// *** Util ***

function getRange(text: string, element: string) {
  if (element.includes("\n"))
    throw new Error(`Multiline element not implemented: ${element}`)

  const lines = text.split("\n")
  const posIndices = lines
    .map(line => line.indexOf(element))
    .filter(index => index !== -1)

  const lineIndex = posIndices.findIndex(pos => pos !== -1)
  if (lineIndex === -1)
    throw new Error(`Element ${element} not found in ${text}`)

  return {
    start: { line: lineIndex, character: posIndices[lineIndex] },
    end: { line: lineIndex, character: posIndices[lineIndex] + element.length }
  }
}

function delay(ms: number) {
  return new Promise((resolve) => {
    setTimeout(resolve, ms)
  })
}

// *** Client ***

interface Message {
	jsonrpc: "2.0"
}

interface RequestMessage extends Message {
  id: number | string
  method: string
  params?: unknown[] | object
}

interface ResponseMessage extends Message {
  id: number | string | null
  result?: any
  error?: any
}

interface NotificationMessage extends Message {
  method: string;
  params?: unknown[] | object
}

type TypeRequestKind = "Default" | "Contextual" | "ContextualCompletions"

/**
 * This is a simplistic client model modelling the IDE which sends to TS-Go both standard LSP requests
 * (didOpen etc) and type requests via the custom `handleCustomTsServerCommand` LSP handler.
 */
class LspClient {
  testProjectDir = path.join(import.meta.dirname, "..", "testProject")

  async spawnAndInit() {
    await this.#openLogFileForWriting()
    this.#spawnSubprocess()
    this.#addStderrHandler()
    this.#addStdoutHandler()
    await this.#sendRequest({
      jsonrpc: "2.0",
      id: process.pid,
      method: "initialize",
      params: {
        processId: process.pid,
        rootPath: this.testProjectDir,
        rootUri: url.pathToFileURL(this.testProjectDir),
        capabilities: {},
        clientInfo: { name: "LSP API Test" },
      },
    })
    this.#sendMessage({jsonrpc: "2.0", method:"initialized", params:{}})
    return this
  }

  #logFileHandle: fs.promises.FileHandle | undefined
  #logFileWritingPromise: Promise<unknown> | undefined
  async #openLogFileForWriting() {
    const logFile = path.join(import.meta.dirname, "..", "tsgo.log")
    this.#logFileHandle = await fs.promises.open(logFile, "w")
  }

  async #log(src: string, msg: string) {
    for ( ; ; ) {
      const promise = this.#logFileWritingPromise
      await promise
      if (promise === this.#logFileWritingPromise) break
    }

    const newline = msg.at(-1) === "\n" ? "" : "\n"
    const writePromise = fs.promises.writeFile(this.#logFileHandle!, Buffer.from(`${src} ${new Date().toISOString()} ${msg}${newline}`))
    this.#logFileWritingPromise = writePromise
    return writePromise
  }

  #childProcess: subprocess.ChildProcess | undefined
  #spawnSubprocess() {
    const tsgoExec = path.join(import.meta.dirname, "..", "..", "..", "built", "local", "tsgo")
    const args = [ "--lsp", "--stdio" ]
    this.#log("---", `Starting ${tsgoExec} with args [${args.join(", ")}]`)
    this.#childProcess = subprocess.execFile(tsgoExec, args)
    this.#log("---", `The subprocess started with pid ${this.#childProcess.pid}`)
  }

  #addStderrHandler() {
    this.#childProcess!.stderr!.on("data", data => {
      this.#log("ERR", String(data))
    })
  }

  #inDataPending = ''
  #addStdoutHandler() {
    this.#childProcess!.stdout!.on("data", newData => {
      this.#log("IN ", String(newData))
      this.#inDataPending = this.#handleInData(this.#inDataPending + newData)
    })
  }

  #handleInData(inData: string) {
    const sep = "\r\n\r\n"
    const sepPos = inData.indexOf(sep)
    if (sepPos === -1) return inData

    const m = /Content-Length: (\d+)\r\n/.exec(inData)
    if (!m) throw new Error(`Content-Length not found in ${inData}`)

    const inDataPendingBuf = Buffer.from(inData) // non-ascii
    const contentLength = Number(m[1])
    const content = inDataPendingBuf.subarray(sepPos + sep.length, sepPos + sep.length + contentLength)

    this.#handleInputMessage(JSON.parse(String(content)))

    const remainingData = String(inDataPendingBuf.subarray(sepPos + sep.length + contentLength))
    return remainingData
  }

  #pendingIdToResponseConsumer = new Map<string | number, {resolve: (r: ResponseMessage) => void, reject: (e: unknown) => void}>()
  #handleInputMessage(message: RequestMessage | ResponseMessage | NotificationMessage) {
    if ('params' in message && 'id' in message) {
      // server -> client request
      this.#sendMessage({jsonrpc: "2.0", id: message.id, result: null})
    } else if ('id' in message && message.id != null) {
      const consumer = this.#pendingIdToResponseConsumer.get(message.id)
      if (consumer) {
        this.#pendingIdToResponseConsumer.delete(message.id)
        if ('error' in message)
          consumer.reject(message)
        else
          consumer.resolve(message)
      }
    } else if ('error' in message) {
      this.#rejectAllPendingRequests(message)
    }
  }

  async #sendRequest(request: RequestMessage) {
    return new Promise((resolve: (r: ResponseMessage) => void, reject: any) => {
      if (this.#pendingIdToResponseConsumer.has(request.id))
        throw new Error(`Duplicate request id ${request.id}`)
      this.#pendingIdToResponseConsumer.set(request.id, {resolve, reject})
      this.#sendMessage(request)
    })
  }

  #sendMessage(request: RequestMessage | ResponseMessage | NotificationMessage) {
    const requestStr = JSON.stringify(request)
    const requestBuf = Buffer.from(requestStr)
    const header = `Content-Length: ${requestBuf.length}\r\n\r\n`
    this.#log("OUT", header)
    this.#log("OUT", requestStr)
    this.#childProcess!.stdin!.write(header)
    this.#childProcess!.stdin!.write(requestBuf)
  }

  #getFileUri(fileRelName: string) {
    return url.pathToFileURL(path.join(this.testProjectDir, fileRelName))
  }

  #_nextId = 0
  #nextId() {
    return this.#_nextId++
  }

  didOpen(fileRelName: string, text: string) {
    this.#sendMessage({
      jsonrpc: "2.0",
      method:"textDocument/didOpen",
      params: {
        textDocument: {
          uri: this.#getFileUri(fileRelName),
          languageId: "typescript",
          version: 0,
          text,
        }
      }
    })
  }

  async getElementType(fileRelName: string, range: Range, forceReturnType: boolean = false, typeRequestKind: TypeRequestKind = "Default") {
    const file = this.#getFileUri(fileRelName)
    const response = await this.#sendRequest({
      jsonrpc: "2.0",
      id: this.#nextId(),
      method: "$/ij/handleCustomTsServerCommand",
      params: {
        ideCommand:"ideGetElementType",
        args: {
          file,
          range,
          forceReturnType,
          typeRequestKind,
        }
      }
    })
    return convertToClientObject(response.result.response, this, file) as Type
  }

  async getTypeProperties(
    typeId: number,
    ideProjectId: number,
    ideTypeCheckerId: number,
    originalRequestUri?: URL,
  ) {
    const response = await this.#sendRequest({
      jsonrpc: "2.0",
      id: this.#nextId(),
      method: "$/ij/handleCustomTsServerCommand",
      params: {
        ideCommand:"ideGetTypeProperties",
        args: { typeId, ideProjectId, ideTypeCheckerId, originalRequestUri},
      }
    })
    return convertToClientObject(response.result.response, this) as Type
  }

  async shutdown() {
    await this.#sendRequest({jsonrpc: "2.0", id: this.#nextId(), method: "shutdown"})
    this.#sendMessage({jsonrpc: "2.0", method: "exit"})
    while (this.#childProcess!.exitCode == null) await delay(50)
    await this.#log("---", `The subprocess exited with code ${this.#childProcess!.exitCode}`)
    await this.#logFileHandle!.close();
    this.#rejectAllPendingRequests(new Error("The subprocess exited"))
  }

  async kill() {
    this.#childProcess!.kill()
    await this.#logFileHandle!.close()
    this.#rejectAllPendingRequests(new Error("The subprocess was killed"))
  }

  #rejectAllPendingRequests(reason: any) {
    for (const {reject} of this.#pendingIdToResponseConsumer.values())
      reject(reason)
    this.#pendingIdToResponseConsumer.clear()
  }
}

// *** Convert types ***

type ServerObjectDef = {ideObjectId: number}
type ServerObjectRef = {ideObjectIdRef: number}

type ServerType = ServerObjectDef & {
  ideObjectType: "TypeObject"
  id: number
  ideProjectId: number
  ideTypeCheckerId: number
}
type ServerSymbol = ServerObjectDef & {
  ideObjectType: "SymbolObject"
  id: number
}

type ServerSignature = ServerObjectDef & {
  ideObjectType: "SignatureObject"
}

type ServerIndexInfo = ServerObjectDef & {
  ideObjectType: "IndexInfo"
}

type ServerNode = ServerObjectDef & {
  ideObjectType: "NodeObject"
}

type ServerOtherObject = ServerObjectDef & {}

type ServerObject = ServerObjectRef | ServerType | ServerSymbol | ServerIndexInfo | ServerNode | ServerOtherObject

/**
 * These are properties which were formerly received via Checker, Type etc methods and properties.
 * Some of them were internal.
 * 
 * They can be moved now to the `handleCustomTsServerCommand` handler.
 */
type TypeEx = Type & {
  /**
   * Type.id
   */
  id: number
  /**
   * Checker.getResolvedTypeArguments()
   */
  resolvedTypeArguments?: Type[]
  /**
   * Checker.getBaseConstraintsOfType()
   */
  constraint?: Type
  /**
   * This is enumQualifiedName received from Nodes structure
   */
  nameType?: string
  /**
   * Property of TypeParameter
   */
  isThisType?: boolean
  /**
   * Property of IntrinsicType
   */
  intrinsicName?: string

  /**
   * Async versions of Type.get...() methods
   */
  getPropertiesEx(): Promise<Symbol[]>
  getCallSignaturesEx(): Promise<Signature[]>
  getConstructSignaturesEx(): Promise<Signature[]>

  /**
   * Cached results of 'getTypeProperties'
   */
  callSignatures?: Signature[]
  constructSignatures?: Signature[]
  indexInfos?: IndexInfo[]
  properties?: Symbol[]
  resolvedFalseType?: Type
  resolvedProperties?: Symbol[]
  resolvedTrueType?: Type
}

type ClientObject = Type | Symbol | Signature | IndexInfo | Node | PseudoBigInt

function convertToClientObject(rootServerObject: ServerObject, lspClient: LspClient, fileUri?: URL): ClientObject {
  const references = new Map<number, ClientObject>()
  function resolveOrConvert<From extends ServerObject, To extends ClientObject>(
    serverObj: From,
    converter: (serverObj: From, clientObj: To) => To,
  ): To {
    const {ideObjectIdRef} = serverObj as ServerObjectRef
    if (ideObjectIdRef != null) {
      const resolvedClientObj = references.get(ideObjectIdRef)
      if (!resolvedClientObj)
        throw new Error(`Could not resolve reference ${ideObjectIdRef} in ${JSON.stringify(serverObj)}`)
      return resolvedClientObj as To
    }

    const {ideObjectId} = serverObj as ServerObjectDef
    if (ideObjectId == null)
      throw new Error(`Neither ideObjectId nor ideObjectIdRef in ${JSON.stringify(serverObj)}`)
    if (references.has(ideObjectId))
      throw new Error(`Duplicate ideObjectId ${ideObjectId} in ${JSON.stringify(serverObj)}`)

    const clientObj = {} as To
    references.set(ideObjectId, clientObj)
    converter(serverObj, clientObj)
    return clientObj
  }

  function convertType(serverType: ServerType, target: Type): Type {

    // Properties returned by both "ideGetElementType" and "getPropertiesOfType"

    if ("flags" in serverType)
      target.flags = serverType.flags as number  
    else
      throw new Error(`Expected "flags" in ${JSON.stringify(serverType)}`)

    if ("id" in serverType && typeof serverType.id === "number")
      (target as TypeEx).id = serverType.id

    if ("objectFlags" in serverType)
      (target as ObjectType).objectFlags = serverType.objectFlags as number

    // Returned by "ideGetElementType" (alphabetically)

    if ("aliasSymbol" in serverType && isSymbol(serverType.aliasSymbol))
      target.aliasSymbol = resolveOrConvert(serverType.aliasSymbol, convertSymbol)

    if ("aliasTypeArguments" in serverType && isTypes(serverType.aliasTypeArguments))
      target.aliasTypeArguments = serverType.aliasTypeArguments
        .map(arg => resolveOrConvert(arg, convertType))

    if ("baseType" in serverType && isType(serverType.baseType))
      (target as SubstitutionType).baseType = resolveOrConvert(serverType.baseType, convertType)

    if ("checkType" in serverType && isType(serverType.checkType))
      (target as ConditionalType).checkType = resolveOrConvert(serverType.checkType, convertType)

    if ("constraint" in serverType && isType(serverType.constraint))
      (target as TypeEx).constraint = resolveOrConvert(serverType.constraint, convertType)

    if ("elementFlags" in serverType && isNumbers(serverType.elementFlags))
      (target as TupleType).elementFlags = serverType.elementFlags

    if ("extendsType" in serverType && isType(serverType.extendsType))
      (target as ConditionalType).extendsType = resolveOrConvert(serverType.extendsType, convertType)

    if ("freshType" in serverType && isType(serverType.freshType))
      (target as LiteralType).freshType = resolveOrConvert(serverType.freshType, convertType) as LiteralType

    if ("indexType" in serverType && isType(serverType.indexType))
      (target as IndexedAccessType).indexType = resolveOrConvert(serverType.indexType, convertType)

    if ("intrinsicName" in serverType && typeof serverType.intrinsicName === "string")
      (target as TypeEx).intrinsicName = serverType.intrinsicName

    if ("isThisType" in serverType && typeof serverType.isThisType === "boolean")
      (target as TypeEx).isThisType = serverType.isThisType

    if ("nameType" in serverType && typeof serverType.nameType === "string")
      (target as TypeEx).nameType = serverType.nameType

    if ("objectType" in serverType && isType(serverType.objectType))
      (target as IndexedAccessType).objectType = resolveOrConvert(serverType.objectType, convertType)

    if ("resolvedTypeArguments" in serverType && isTypes(serverType.resolvedTypeArguments))
      (target as TypeEx).resolvedTypeArguments = serverType.resolvedTypeArguments
        .map(t => resolveOrConvert(t, convertType))

    if ("symbol" in serverType && isSymbol(serverType.symbol))
      target.symbol = resolveOrConvert(serverType.symbol, convertSymbol)

    if ("target" in serverType && isType(serverType.target))
      (target as TypeReference).target = resolveOrConvert(serverType.target, convertType) as GenericType

    if ("texts" in serverType && isStrings(serverType.texts))
      (target as TemplateLiteralType).texts = serverType.texts

    if ("type" in serverType && isType(serverType.type))
      (target as IndexType).type = resolveOrConvert(serverType.type, convertType)

    if ("types" in serverType && isTypes(serverType.types))
      (target as UnionType).types = serverType.types
        .map(t => resolveOrConvert(t, convertType))

    if ("value" in serverType) {
      if (isOtherObject(serverType.value))
        (target as BigIntLiteralType).value = resolveOrConvert(serverType.value, convertPseudoBigInt)
      if (typeof serverType.value === "string" || typeof serverType.value === "number")
        (target as LiteralType).value = serverType.value
    }

    // Returned by "getPropertiesOfType"

    if ("callSignatures" in serverType && isSignatures(serverType.callSignatures))
      (target as TypeEx).callSignatures = serverType.callSignatures
        .map(sig => resolveOrConvert(sig, convertSignature))

    if ("constructSignatures" in serverType && isSignatures(serverType.constructSignatures))
      (target as TypeEx).constructSignatures = serverType.constructSignatures
        .map(sig => resolveOrConvert(sig, convertSignature))

    if ("properties" in serverType && isSymbols(serverType.properties))
      (target as TypeEx).properties = serverType.properties
        .map(sym => resolveOrConvert(sym, convertSymbol))

    if ("resolvedFalseType" in serverType && isType(serverType.resolvedFalseType))
      (target as TypeEx).resolvedFalseType = resolveOrConvert(serverType.resolvedFalseType, convertType)

    if ("resolvedProperties" in serverType && isSymbols(serverType.resolvedProperties))
      (target as TypeEx).resolvedProperties = serverType.resolvedProperties
        .map(sym => resolveOrConvert(sym, convertSymbol))

    if ("resolvedTrueType" in serverType && isType(serverType.resolvedTrueType))
      (target as TypeEx).resolvedTrueType = resolveOrConvert(serverType.resolvedTrueType, convertType)

    if ("indexInfos" in serverType && isIndexInfos(serverType.indexInfos))
      (target as TypeEx).indexInfos = serverType.indexInfos
        .map(info => resolveOrConvert(info, convertIndexInfo))

    // Methods

    target.getFlags = () => target.flags
    target.getSymbol = () => target.symbol

    let typePropertiesResponse: Type | undefined
    async function getTypeProperties() {
      if (!typePropertiesResponse)
        typePropertiesResponse = await lspClient.getTypeProperties(
          serverType.id,
          serverType.ideProjectId,
          serverType.ideTypeCheckerId,
          fileUri,
        )
      return typePropertiesResponse as TypeEx
    }
    (target as TypeEx).getPropertiesEx = async () => (await getTypeProperties()).properties!;
    (target as TypeEx).getCallSignaturesEx = async () => (await getTypeProperties()).callSignatures!;
    (target as TypeEx).getConstructSignaturesEx = async () => (await getTypeProperties()).constructSignatures!

    return target
  }

  function convertSymbol(symbolServerObj: ServerSymbol, target: Symbol): Symbol {
    // wip
    return target
  }

  function convertSignature(signatureServerObj: ServerSignature, target: Signature): Signature {
    // wip
    return target
  }

  function convertIndexInfo(indexInfoServerObj: ServerIndexInfo, target: IndexInfo): IndexInfo {
    // wip
    return target
  }

  function convertNode(nodeServerObj: ServerNode, target: Node): Node {
    // wip
    return target
  }

  function convertPseudoBigInt(pseudoBigIntServerObj: ServerOtherObject, target: PseudoBigInt): PseudoBigInt {
    // wip
    return target
  }

  function isType(serverObject: unknown): serverObject is ServerType {
    return (serverObject as ServerType).ideObjectType === "TypeObject"
  }

  function isTypes(serverObject: unknown): serverObject is ServerType[] {
    return Array.isArray(serverObject) && serverObject.every(isType)
  }

  function isSymbol(serverObject: unknown): serverObject is ServerSymbol {
    return (serverObject as ServerSymbol).ideObjectType === "SymbolObject"
  }

  function isSymbols(serverObject: unknown): serverObject is ServerSymbol[] {
    return Array.isArray(serverObject) && serverObject.every(isSymbol)
  }

  function isSignature(serverObject: unknown): serverObject is ServerSignature {
    return (serverObject as ServerSignature).ideObjectType === "SignatureObject"
  }

  function isSignatures(serverObject: unknown): serverObject is ServerSignature[] {
    return Array.isArray(serverObject) && serverObject.every(isSignature)
  }

  function isIndexInfo(serverObject: unknown): serverObject is ServerIndexInfo {
    return (serverObject as ServerIndexInfo).ideObjectType === "IndexInfo"
  }

  function isIndexInfos(serverObject: unknown): serverObject is ServerIndexInfo[] {
    return Array.isArray(serverObject) && serverObject.every(isIndexInfo)
  }

  function isNode(serverObject: unknown): serverObject is ServerNode {
    return (serverObject as ServerNode).ideObjectType === "NodeObject"
  }

  function isOtherObject(serverObject: unknown): serverObject is ServerOtherObject {
    return (serverObject as ServerObjectDef).ideObjectId != null
      && !isType(serverObject)
      && !isSymbol(serverObject)
      && !isNode(serverObject)
  }

  function isStrings(serverObject: unknown): serverObject is string[] {
    return Array.isArray(serverObject) && serverObject.every(s => typeof s === "string")
  }

  function isNumbers(serverObject: unknown): serverObject is number[] {
    return Array.isArray(serverObject) && serverObject.every(n => typeof n === "number")
  }


  if (isType(rootServerObject))
    return resolveOrConvert(rootServerObject, convertType)

  if (isSymbol(rootServerObject))
    return resolveOrConvert(rootServerObject, convertSymbol)

  if (isNode(rootServerObject))   
    return resolveOrConvert(rootServerObject, convertNode)


  throw new Error(`Unexpected rootServerObject ${JSON.stringify(rootServerObject)}`)
}
