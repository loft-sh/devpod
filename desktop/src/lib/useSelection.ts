import { useCallback, useMemo, useState } from "react"

export function useSelection<TIdType extends string | number = string>() {
  const [selectedItems, setSelectedItems] = useState(new Set<TIdType>())

  const toggleSelection = useCallback(
    (id: TIdType) => {
      setSelectedItems((curr) => {
        const updated = new Set(curr)
        if (updated.has(id)) {
          updated.delete(id)
        } else {
          updated.add(id)
        }

        return updated
      })
    },
    [setSelectedItems]
  )

  const toggleSelectAll = useCallback(
    (allSet: TIdType[]) => {
      setSelectedItems((curr) => {
        if (curr.size === allSet.length) {
          return new Set<TIdType>()
        }

        return new Set<TIdType>(allSet)
      })
    },
    [setSelectedItems]
  )

  const setSelected = useCallback(
    (id: TIdType, selected: boolean) => {
      if (!selectedItems.has(id) && selected) {
        const updated = new Set<TIdType>(selectedItems)
        updated.add(id)
        setSelectedItems(updated)
      } else if (selectedItems.has(id) && !selected) {
        const updated = new Set<TIdType>(selectedItems)
        updated.delete(id)
        setSelectedItems(updated)
      }
    },
    [setSelectedItems, selectedItems]
  )

  // Will remove outdated selection items if they are no longer part of the entire set.
  const prune = useCallback(
    (allSet: TIdType[]) => {
      const updated = new Set(selectedItems)

      let changed = false

      for (const id of selectedItems) {
        if (!allSet.includes(id)) {
          updated.delete(id)
          changed = true
        }
      }

      if (changed) {
        setSelectedItems(updated)
      }
    },
    [setSelectedItems, selectedItems]
  )

  const clear = useCallback(() => {
    setSelectedItems(new Set<TIdType>())
  }, [setSelectedItems])

  const has = useCallback(
    (id: TIdType) => {
      return selectedItems.has(id)
    },
    [selectedItems]
  )

  return useMemo(
    () => ({
      toggleSelection,
      toggleSelectAll,
      size: selectedItems.size,
      clear,
      prune,
      setSelected,
      has,
      selectedItems: {
        val: selectedItems,
        set: setSelectedItems,
      },
    }),
    [
      selectedItems,
      setSelectedItems,
      toggleSelection,
      toggleSelectAll,
      clear,
      prune,
      has,
      setSelected,
    ]
  )
}
