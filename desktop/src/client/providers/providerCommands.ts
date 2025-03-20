import { exists, getErrorFromChildProcess, Result, ResultError, Return } from "../../lib"
import {
  TAddProviderConfig,
  TCheckProviderUpdateResult,
  TProviderID,
  TProviderOptions,
  TProviders,
  TProviderSource,
} from "../../types"
import { Command, isOk, serializeRawOptions, toFlagArg } from "../command"
import {
  DEVPOD_COMMAND_ADD,
  DEVPOD_COMMAND_DELETE,
  DEVPOD_COMMAND_GET_PROVIDER_NAME,
  DEVPOD_COMMAND_LIST,
  DEVPOD_COMMAND_OPTIONS,
  DEVPOD_COMMAND_PROVIDER,
  DEVPOD_COMMAND_SET_OPTIONS,
  DEVPOD_COMMAND_UPDATE,
  DEVPOD_COMMAND_USE,
  DEVPOD_FLAG_DEBUG,
  DEVPOD_FLAG_DRY,
  DEVPOD_FLAG_JSON_LOG_OUTPUT,
  DEVPOD_FLAG_JSON_OUTPUT,
  DEVPOD_FLAG_NAME,
  DEVPOD_FLAG_RECONFIGURE,
  DEVPOD_FLAG_SINGLE_MACHINE,
  DEVPOD_FLAG_USE,
} from "../constants"
import { DEVPOD_COMMAND_CHECK_PROVIDER_UPDATE, DEVPOD_COMMAND_HELPER } from "./../constants"

export class ProviderCommands {
  static DEBUG = false

  private static newCommand(args: string[]): Command {
    return new Command([...args, ...(ProviderCommands.DEBUG ? [DEVPOD_FLAG_DEBUG] : [])])
  }

  static async ListProviders(): Promise<Result<TProviders>> {
    const result = await new Command([
      DEVPOD_COMMAND_PROVIDER,
      DEVPOD_COMMAND_LIST,
      DEVPOD_FLAG_JSON_OUTPUT,
      DEVPOD_FLAG_JSON_LOG_OUTPUT,
    ]).run()
    if (result.err) {
      return result
    }

    if (!isOk(result.val)) {
      return getErrorFromChildProcess(result.val)
    }

    const rawProviders = JSON.parse(result.val.stdout) as TProviders
    for (const provider of Object.values(rawProviders)) {
      provider.isProxyProvider =
        provider.config?.exec?.proxy !== undefined || provider.config?.exec?.daemon !== undefined
    }

    return Return.Value(rawProviders)
  }

  static async GetProviderID(source: string) {
    const result = await new Command([
      DEVPOD_COMMAND_HELPER,
      DEVPOD_COMMAND_GET_PROVIDER_NAME,
      source,
      DEVPOD_FLAG_JSON_LOG_OUTPUT,
    ]).run()
    if (result.err) {
      return result
    }

    if (!isOk(result.val)) {
      return getErrorFromChildProcess(result.val)
    }

    return Return.Value(result.val.stdout)
  }

  static async AddProvider(
    rawProviderSource: string,
    config: TAddProviderConfig
  ): Promise<ResultError> {
    const maybeName = config.name
    const maybeNameFlag = exists(maybeName) ? [toFlagArg(DEVPOD_FLAG_NAME, maybeName)] : []
    const useFlag = toFlagArg(DEVPOD_FLAG_USE, "false")

    const result = await ProviderCommands.newCommand([
      DEVPOD_COMMAND_PROVIDER,
      DEVPOD_COMMAND_ADD,
      rawProviderSource,
      ...maybeNameFlag,
      useFlag,
      DEVPOD_FLAG_JSON_LOG_OUTPUT,
    ]).run()
    if (result.err) {
      return result
    }

    if (!isOk(result.val)) {
      return getErrorFromChildProcess(result.val)
    }

    return Return.Ok()
  }

  static async RemoveProvider(id: TProviderID) {
    const result = await ProviderCommands.newCommand([
      DEVPOD_COMMAND_PROVIDER,
      DEVPOD_COMMAND_DELETE,
      id,
      DEVPOD_FLAG_JSON_LOG_OUTPUT,
    ]).run()
    if (result.err) {
      return result
    }

    if (!isOk(result.val)) {
      return getErrorFromChildProcess(result.val)
    }

    return Return.Ok()
  }

  static async UseProvider(
    id: TProviderID,
    rawOptions?: Record<string, unknown>,
    reuseMachine?: boolean
  ) {
    const optionsFlag = rawOptions ? serializeRawOptions(rawOptions) : []
    const maybeResuseMachineFlag = reuseMachine ? [DEVPOD_FLAG_SINGLE_MACHINE] : []

    const result = await ProviderCommands.newCommand([
      DEVPOD_COMMAND_PROVIDER,
      DEVPOD_COMMAND_USE,
      id,
      ...optionsFlag,
      ...maybeResuseMachineFlag,
      DEVPOD_FLAG_JSON_LOG_OUTPUT,
    ]).run()

    if (result.err) {
      return result
    }

    if (!isOk(result.val)) {
      return getErrorFromChildProcess(result.val)
    }

    return Return.Ok()
  }

  static async SetProviderOptions(
    id: TProviderID,
    rawOptions: Record<string, unknown>,
    reuseMachine: boolean,
    dry?: boolean,
    reconfigure?: boolean
  ) {
    const optionsFlag = serializeRawOptions(rawOptions)
    const maybeResuseMachineFlag = reuseMachine ? [DEVPOD_FLAG_SINGLE_MACHINE] : []
    const maybeDry = dry ? [DEVPOD_FLAG_DRY] : []
    const maybeReconfigure = reconfigure ? [DEVPOD_FLAG_RECONFIGURE] : []

    const result = await ProviderCommands.newCommand([
      DEVPOD_COMMAND_PROVIDER,
      DEVPOD_COMMAND_SET_OPTIONS,
      id,
      ...optionsFlag,
      ...maybeResuseMachineFlag,
      ...maybeDry,
      ...maybeReconfigure,
      DEVPOD_FLAG_JSON_LOG_OUTPUT,
    ]).run()
    if (result.err) {
      return result
    }

    if (!isOk(result.val)) {
      return getErrorFromChildProcess(result.val)
    } else if (dry) {
      return Return.Value(JSON.parse(result.val.stdout) as TProviderOptions)
    }

    return Return.Ok()
  }

  static async GetProviderOptions(id: TProviderID) {
    const result = await new Command([
      DEVPOD_COMMAND_PROVIDER,
      DEVPOD_COMMAND_OPTIONS,
      id,
      DEVPOD_FLAG_JSON_OUTPUT,
      DEVPOD_FLAG_JSON_LOG_OUTPUT,
    ]).run()
    if (result.err) {
      return result
    }

    if (!isOk(result.val)) {
      return getErrorFromChildProcess(result.val)
    }

    return Return.Value(JSON.parse(result.val.stdout) as TProviderOptions)
  }

  static async CheckProviderUpdate(id: TProviderID) {
    const result = await new Command([
      DEVPOD_COMMAND_HELPER,
      DEVPOD_COMMAND_CHECK_PROVIDER_UPDATE,
      id,
    ]).run()
    if (result.err) {
      return result
    }

    if (!isOk(result.val)) {
      return getErrorFromChildProcess(result.val)
    }

    return Return.Value(JSON.parse(result.val.stdout) as TCheckProviderUpdateResult)
  }

  static async UpdateProvider(id: TProviderID, source: TProviderSource) {
    const useFlag = toFlagArg(DEVPOD_FLAG_USE, "false")

    const result = await new Command([
      DEVPOD_COMMAND_PROVIDER,
      DEVPOD_COMMAND_UPDATE,
      id,
      source.raw ?? source.github ?? source.url ?? source.file ?? "",
      DEVPOD_FLAG_JSON_LOG_OUTPUT,
      useFlag,
    ]).run()
    if (result.err) {
      return result
    }

    if (!isOk(result.val)) {
      return getErrorFromChildProcess(result.val)
    }

    return Return.Ok()
  }
}
