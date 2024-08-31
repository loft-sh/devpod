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

export function useProviderDisplayOptions(
  options: TProviderOptions | undefined,
  optionGroups: TProviderOptionGroup[],
  edit?: boolean
): TAllOptions {
  return useMemo(
    () => processDisplayOptions(options, optionGroups, edit),
    [optionGroups, options, edit]
  )
}

// processProviderOptions reads, parses and groups all options into a displayable format.
// Option groups can contain wildcard options `MY_PREFIX_*`. All options matching the wildcard will be added to this group.
// The first group to claim a wildcard takes precendence.
export function processDisplayOptions(
  options: TProviderOptions | undefined,
  optionGroups: TProviderOptionGroup[],
  edit?: boolean
): TAllOptions {
  const initialGroups = optionGroups.map((group) => ({ ...group, options: [] }))
  const empty: TAllOptions = { required: [], groups: initialGroups, other: [] }
  if (!exists(options)) {
    return empty
  }

  return getVisibleOptions(options, edit).reduce<TAllOptions>((acc, option) => {
    const optionGroup = optionGroups.find((group) => {
      return group.options?.find((o) => optionMatches(o, option.id))
    })

    if (optionGroup && optionGroup.name) {
      // create group if not found
      const groupIdx = acc.groups.findIndex((g) => g.name === optionGroup.name)
      if (groupIdx === -1) {
        acc.groups.push({
          name: optionGroup.name,
          options: [],
          defaultVisible: optionGroup.defaultVisible,
        })
        acc.groups[acc.groups.length - 1]?.options.push(option)

        return acc
      }

      acc.groups[groupIdx]?.options.push(option)

      return acc
    }

    if (option.required) {
      acc.required.push(option)

      return acc
    }

    acc.other.push(option)

    return acc
  }, empty)
}

function optionMatches(optionName: string, optionID: string): boolean {
  if (optionName.includes("*")) {
    const regEx = new RegExp("^" + optionName.replaceAll("*", ".*") + "$")

    return regEx.test(optionID)
  }

  return optionName === optionID
}
