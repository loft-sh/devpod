import {
  Button,
  Menu,
  MenuButton,
  MenuItemOption,
  MenuList,
  MenuOptionGroup,
} from "@chakra-ui/react"
import { ChevronDownIcon } from "@chakra-ui/icons"
import { DEFAULT_SORT_WORKSPACE_MODE, ESortWorkspaceMode } from "@/lib/useSortWorkspaces"
import { useCallback } from "react"

export function WorkspaceSorter({
  sortMode,
  setSortMode,
}: {
  sortMode: ESortWorkspaceMode
  setSortMode: (sortMode: ESortWorkspaceMode) => void
}) {
  const onChange = useCallback(
    (value: string | string[] | undefined) => {
      const mode = Array.isArray(value)
        ? (value[0] as ESortWorkspaceMode | undefined)
        : (value as ESortWorkspaceMode | undefined)
      setSortMode(mode ?? DEFAULT_SORT_WORKSPACE_MODE)
    },
    [setSortMode]
  )

  return (
    <Menu offset={[0, 2]}>
      <MenuButton as={Button} variant="outline" rightIcon={<ChevronDownIcon boxSize={4} />}>
        Sort by: {sortMode}
      </MenuButton>
      <MenuList>
        <MenuOptionGroup type="radio" value={sortMode} onChange={onChange}>
          {Object.values(ESortWorkspaceMode).map((option) => (
            <MenuItemOption key={option} value={option}>
              {option}
            </MenuItemOption>
          ))}
        </MenuOptionGroup>
      </MenuList>
    </Menu>
  )
}
