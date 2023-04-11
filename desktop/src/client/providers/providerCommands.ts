import { exists, Result, ResultError, Return, getErrorFromChildProcess } from "../../lib"
import { TAddProviderConfig, TProviderID, TProviderOptions, TProviders } from "../../types"
import { Command, isOk, toFlagArg } from "../command"
import {
  DEVPOD_COMMAND_ADD,
  DEVPOD_COMMAND_DELETE,
  DEVPOD_COMMAND_GET_PROVIDER_NAME,
  DEVPOD_COMMAND_HELPER,
  DEVPOD_COMMAND_INIT,
  DEVPOD_COMMAND_LIST,
  DEVPOD_COMMAND_OPTIONS,
  DEVPOD_COMMAND_PROVIDER,
  DEVPOD_COMMAND_SET_OPTIONS,
  DEVPOD_COMMAND_USE,
  DEVPOD_FLAG_DEBUG,
  DEVPOD_FLAG_JSON_LOG_OUTPUT,
  DEVPOD_FLAG_JSON_OUTPUT,
  DEVPOD_FLAG_NAME,
  DEVPOD_FLAG_OPTION,
  DEVPOD_FLAG_SINGLE_MACHINE,
  DEVPOD_FLAG_USE,
} from "../constants"

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
    ]).run()
    if (result.err) {
      return result
    }

    const rawProviders = JSON.parse(result.val.stdout) as TProviders

    return Return.Value(rawProviders)
  }

  static async GetProviderID(source: string) {
    const result = await new Command([
      DEVPOD_COMMAND_HELPER,
      DEVPOD_COMMAND_GET_PROVIDER_NAME,
      source,
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
    const optionsFlag = rawOptions
      ? [toFlagArg(DEVPOD_FLAG_OPTION, serializeRawOptions(rawOptions))]
      : []
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
    reuseMachine: boolean
  ) {
    const optionsFlag = toFlagArg(DEVPOD_FLAG_OPTION, serializeRawOptions(rawOptions))
    const maybeResuseMachineFlag = reuseMachine ? [DEVPOD_FLAG_SINGLE_MACHINE] : []

    const result = await ProviderCommands.newCommand([
      DEVPOD_COMMAND_PROVIDER,
      DEVPOD_COMMAND_SET_OPTIONS,
      id,
      optionsFlag,
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

  static async GetProviderOptions(id: TProviderID) {
    const result = await new Command([
      DEVPOD_COMMAND_PROVIDER,
      DEVPOD_COMMAND_OPTIONS,
      id,
      DEVPOD_FLAG_JSON_OUTPUT,
    ]).run()
    if (result.err) {
      return result
    }

    if (!isOk(result.val)) {
      return getErrorFromChildProcess(result.val)
    }

    return Return.Value(JSON.parse(result.val.stdout) as TProviderOptions)
  }

  static async InitProvider(id: TProviderID) {
    const result = await ProviderCommands.newCommand([
      DEVPOD_COMMAND_PROVIDER,
      DEVPOD_COMMAND_INIT,
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
}

export function serializeRawOptions(rawOptions: Record<string, unknown>): string {
  return Object.entries(rawOptions)
    .map(([key, value]) => `${key}=${value}`)
    .join(",")
}
