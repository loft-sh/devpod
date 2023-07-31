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
  return exists(providerConfig?.exec?.["create"])
}

function getOptionDisplayName(id: TOptionID) {
  return id
    .toLowerCase()
    .replace(/_/g, " ")
    .replace(/\b\w/g, (l) => l.toUpperCase())
}

export function getVisibleOptions(
  options: TProviderOptions | undefined | null
): readonly TOptionWithID[] {
  return Object.entries(options ?? {})
    .filter(([, { hidden }]) => !(exists(hidden) && hidden))
    .map<TOptionWithID>(([optionName, option]) => ({
      id: optionName,
      defaultValue: option.default,
      displayName: getOptionDisplayName(optionName),
      ...option,
    }))
}
