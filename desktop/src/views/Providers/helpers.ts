import { exists } from "../../lib"
import { TOptionID, TProviderOption, TProviderOptions } from "../../types"

export type TOptionWithID = Readonly<{
  id: TOptionID
  defaultValue: TProviderOption["default"]
  displayName: string
}> &
  Omit<TProviderOption, "default">
export function getOptionValue(option: TOptionWithID) {
  return option.value ?? option.defaultValue
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
