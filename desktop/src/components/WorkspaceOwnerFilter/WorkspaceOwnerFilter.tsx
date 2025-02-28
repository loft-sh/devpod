import { ChevronDownIcon } from "@chakra-ui/icons"
import {
  Button,
  Menu,
  MenuButton,
  MenuItemOption,
  MenuList,
  MenuOptionGroup,
} from "@chakra-ui/react"
import { useCallback } from "react"

export type TWorkspaceOwnerFilterState = "user" | "all"

export function WorkspaceOwnerFilter({
  ownerFilter,
  setOwnerFilter,
}: {
  ownerFilter: TWorkspaceOwnerFilterState
  setOwnerFilter: (ownerFilter: TWorkspaceOwnerFilterState) => void
}) {
  const onChange = useCallback(
    (value: string[] | string) => {
      setOwnerFilter((Array.isArray(value) ? value[0] : value) as TWorkspaceOwnerFilterState)
    },
    [setOwnerFilter]
  )

  return (
    <Menu offset={[0, 2]}>
      <MenuButton as={Button} variant="outline" rightIcon={<ChevronDownIcon boxSize={4} />}>
        Workspaces: {ownerFilter == "user" ? "Mine" : "All"}
      </MenuButton>
      <MenuList>
        <MenuOptionGroup type="radio" value={ownerFilter} onChange={onChange}>
          <MenuItemOption key="user" value="user">
            Mine
          </MenuItemOption>
          <MenuItemOption key="all" value="all">
            All
          </MenuItemOption>
        </MenuOptionGroup>
      </MenuList>
    </Menu>
  )
}
