import { TProviderOptions } from "../../../types"

export const ALLOWED_NAMES_REGEX = /^[a-z0-9\\-]+$/

export function mergeProviderOptions(
  old: TProviderOptions | undefined | null,
  current: TProviderOptions
): TProviderOptions {
  let mergedOptions: TProviderOptions = {}
  if (!old) {
    mergedOptions = current
  } else {
    for (const [optionName, optionValue] of Object.entries(current)) {
      const maybeOption = old[optionName]
      mergedOptions[optionName] = { ...optionValue, value: maybeOption?.value ?? optionValue.value }
    }
  }

  return mergedOptions
}
