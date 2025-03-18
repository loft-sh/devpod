import { TIDE } from "@/types"
import { getIDEDisplayName, useHover } from "@/lib"
import {
  HStack,
  MenuItem,
  PlacementWithLogical,
  Popover,
  PopoverContent,
  PopoverTrigger,
  Portal,
  Text,
  useColorModeValue,
} from "@chakra-ui/react"
import { ChevronRightIcon } from "@chakra-ui/icons"
import { IDEIcon } from "@/components"
import { useCallback, useEffect } from "react"

export function IDEGroup({
  ides,
  group,
  disabled,
  onItemClick,
  placement,
  offset,
  onHoverChange,
}: {
  ides?: TIDE[]
  group: string
  disabled?: boolean
  onItemClick: (ide: TIDE["name"]) => void
  placement?: PlacementWithLogical
  offset?: [number, number]
  onHoverChange?: (group: string, hover: boolean) => void
}) {
  const [popoverHover, popoverRef] = useHover()
  const [triggerHover, triggerRef] = useHover()

  useEffect(() => {
    onHoverChange?.(group, popoverHover || triggerHover)
  }, [popoverHover, triggerHover, onHoverChange, group])

  return (
    <Popover
      isOpen={popoverHover || triggerHover}
      placement={placement ?? "right-end"}
      offset={offset ?? [100, 0]}>
      <PopoverTrigger>
        <MenuItem ref={triggerRef} isDisabled={disabled}>
          <HStack width="full" justifyContent="space-between">
            <Text>{group}</Text>
            <ChevronRightIcon boxSize={4} />
          </HStack>
        </MenuItem>
      </PopoverTrigger>
      <Portal>
        <PopoverContent zIndex="popover" width="fit-content" ref={popoverRef}>
          {ides?.map((ide) => (
            <IDEItem key={ide.name} ide={ide} onItemClick={onItemClick} disabled={disabled} />
          ))}
        </PopoverContent>
      </Portal>
    </Popover>
  )
}

function IDEItem({
  disabled,
  onItemClick,
  ide,
}: {
  ide: TIDE
  onItemClick: (ide: TIDE["name"]) => void
  disabled?: boolean
}) {
  const menuHoverColor = useColorModeValue("gray.100", "gray.700")


  const onClick = useCallback(() => {
    onItemClick(ide.name)
  }, [onItemClick, ide])

  return (
    <MenuItem
      _hover={{ bg: menuHoverColor }}
      isDisabled={disabled}
      onClick={onClick}
      value={ide.name!}
      icon={<IDEIcon ide={ide} width={6} height={6} size="sm" />}>
      {getIDEDisplayName(ide)}
    </MenuItem>
  )
}
