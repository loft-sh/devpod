import { exists } from "../../lib"
import { TOptionID, TProviderConfig, TProviderOption, TProviderOptions } from "../../types"

export type TOptionWithID = Readonly<{
  id: TOptionID
  defaultValue: TProviderOption["default"]
  displayName: string
}> &
  Omit<TProviderOption, "default" | "children">
export function getOptionValue(option: TOptionWithID) {
  return option.value ?? option.defaultValue
}
export function canCreateMachine(providerConfig: TProviderConfig | undefined | null): boolean {
  const exec = providerConfig?.exec
  if (exec?.proxy) {
    return false
  }

  return exists((exec as Record<string, unknown> | undefined)?.["create"])
}

function getOptionFallbackDisplayName(id: TOptionID) {
  return id
    .toLowerCase()
    .replace(/_/g, " ")
    .replace(/\b\w/g, (l) => l.toUpperCase())
}

export function getVisibleOptions(
  options: TProviderOptions | undefined | null,
  edit?: boolean
): readonly TOptionWithID[] {
  return Object.entries(options ?? {})
    .filter(([, { hidden, mutable }]) => {
      if (exists(hidden) && hidden) {
        return false
      }
      if (edit && !mutable) {
        return false
      }

      return true
    })
    .map<TOptionWithID>(([optionName, option]) => ({
      id: optionName,
      defaultValue: option.default,
      ...option,
      displayName: option.displayName || getOptionFallbackDisplayName(optionName),
    }))
}

export function mergeOptionDefinitions(
  stateOptions: TProviderOptions,
  configOptions: TProviderOptions
): TProviderOptions {
  const res: TProviderOptions = {}
  for (const [k, v] of Object.entries(stateOptions)) {
    const config = configOptions[k]
    if (config) {
      res[k] = { ...config, ...v }
      continue
    }

    res[k] = v
  }

  return res
}
