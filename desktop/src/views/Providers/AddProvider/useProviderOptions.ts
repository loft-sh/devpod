import { useMemo } from "react"
import { TOptionWithID, getVisibleOptions } from "../helpers"
import { exists } from "@/lib"
import { TProviderOptionGroup, TProviderOptions } from "@/types"

type TOptionGroup = Readonly<{ options: TOptionWithID[] }> & Omit<TProviderOptionGroup, "options">
type TAllOptions = Readonly<{
  required: TOptionWithID[]
  groups: TOptionGroup[]
  other: TOptionWithID[]
}>

export function useProviderOptions(
  options: TProviderOptions | undefined,
  optionGroups: TProviderOptionGroup[]
) {
  return useMemo(() => {
    const initialGroups = optionGroups.map((group) => ({ ...group, options: [] }))
    const empty: TAllOptions = { required: [], groups: initialGroups, other: [] }
    if (!exists(options)) {
      return empty
    }

    return getVisibleOptions(options).reduce<TAllOptions>((acc, option) => {
      const optionGroup = optionGroups.find((group) => {
        return group.options?.find((o) => optionMatches(o, option.id))
      })

      if (optionGroup && optionGroup.name) {
        const groupIdx = acc.groups.findIndex((g) => g.name === optionGroup.name)
        // create group if not found
        if (groupIdx === -1) {
          acc.groups.push({
            name: optionGroup.name,
            options: [],
            defaultVisible: optionGroup.defaultVisible,
          })
          acc.groups[acc.groups.length - 1]?.options.push(option)
        } else {
          acc.groups[groupIdx]?.options.push(option)
        }

        return acc
      }

      if (option.required) {
        acc.required.push(option)

        return acc
      }

      acc.other.push(option)

      return acc
    }, empty)
  }, [optionGroups, options])
}

function optionMatches(optionName: string, optionID: string): boolean {
  if (optionName.includes("*")) {
    const regEx = new RegExp("^" + optionName.replaceAll("*", ".*") + "$")

    return regEx.test(optionID)
  }

  return optionName === optionID
}
