import { ConfigureProviderOptionsForm } from "@/views/Providers/AddProvider"
import { useProviderOptions } from "@/views/Providers/AddProvider/useProviderOptions"
import { TOptionWithID } from "@/views/Providers/helpers"
import {
  ButtonGroup,
  Heading,
  Icon,
  IconButton,
  List,
  ListItem,
  Popover,
  PopoverArrow,
  PopoverBody,
  PopoverContent,
  PopoverHeader,
  PopoverTrigger,
  Portal,
  Text,
  Tooltip,
  useColorModeValue,
  useDisclosure,
  VStack,
} from "@chakra-ui/react"
import { ReactElement, useRef, useState } from "react"
import { HiPencil } from "react-icons/hi2"
import { TNamedProvider } from "../../../types"
import { ViewIcon } from "@chakra-ui/icons"

type TProviderOptionsPopoverProps = Readonly<{ provider: TNamedProvider; trigger: ReactElement }>
export function ProviderOptionsPopover({ provider, trigger }: TProviderOptionsPopoverProps) {
  const { isOpen, onClose, onOpen } = useDisclosure()
  const bodyRef = useRef<HTMLDivElement>(null)
  const options = useProviderOptions(
    provider?.state?.options ?? {},
    provider?.config?.optionGroups ?? []
  )
  const [isEditing, setIsEditing] = useState(false)
  const reset = () => {
    onClose()
    setIsEditing(false)
  }

  return (
    <Popover
      onClose={reset}
      onOpen={onOpen}
      trigger={isOpen ? "click" : "hover"}
      placement="top"
      isLazy>
      <PopoverTrigger>{trigger}</PopoverTrigger>
      <Portal>
        <PopoverContent minWidth={isEditing ? "2xl" : "sm"} height="full">
          <PopoverArrow />
          <PopoverHeader fontWeight="semibold">
            <VStack align="start" spacing="0">
              <Heading size="sm" as="h3">
                {provider.name}
              </Heading>
              <Text fontSize="xs">Current provider configuration</Text>
            </VStack>
            <ButtonGroup variant="outline">
              <Tooltip label={!isEditing ? "Edit Provider" : "View Configuration"}>
                <IconButton
                  aria-label={!isEditing ? "Edit Provider" : "View Configuration"}
                  onClick={() => setIsEditing((x) => !x)}
                  icon={!isEditing ? <Icon as={HiPencil} boxSize={5} /> : <ViewIcon boxSize={5} />}
                />
              </Tooltip>
            </ButtonGroup>
          </PopoverHeader>
          <PopoverBody
            ref={bodyRef}
            height="full"
            maxHeight={isEditing ? "md" : "sm"}
            overflowX="hidden"
            overflowY="auto"
            paddingBottom="0">
            {!isEditing ? (
              <VStack align="start" paddingTop="2" paddingBottom="4" width="full">
                {options.required.length > 0 && (
                  <>
                    <Heading size="sm">Required</Heading>
                    <ProviderOptionList options={options.required} />
                  </>
                )}

                {options.groups.map(
                  (group) =>
                    group.options.length > 0 && (
                      <>
                        <Heading size="sm">{group.name}</Heading>
                        <ProviderOptionList options={group.options} />
                      </>
                    )
                )}

                {options.other.length > 0 && (
                  <>
                    <Heading size="sm">Other</Heading>
                    <ProviderOptionList options={options.other} />
                  </>
                )}
              </VStack>
            ) : (
              <ConfigureProviderOptionsForm
                isModal
                containerRef={bodyRef}
                providerID={provider.name}
                reuseMachine={!!provider.state?.singleMachine}
                isDefault={!!provider.default}
              />
            )}
          </PopoverBody>
        </PopoverContent>
      </Portal>
    </Popover>
  )
}

type TProviderOptionListProps = Readonly<{ options: readonly TOptionWithID[] }>

function ProviderOptionList({ options }: TProviderOptionListProps) {
  const valueColor = useColorModeValue("gray.500", "gray.400")
  const hoverBackgroundColor = useColorModeValue("white", "gray.700")

  return (
    <List width="full" marginBottom="2">
      {options.map((option) => (
        <ListItem width="full" display="flex" flexFlow="row nowrap">
          <Tooltip label={option.displayName}>
            <Text
              whiteSpace="nowrap"
              wordBreak="keep-all"
              marginRight="4"
              width="28"
              minWidth="32"
              overflowX="hidden"
              textOverflow="ellipsis">
              {option.displayName}
            </Text>
          </Tooltip>
          <Text
            textOverflow="ellipsis"
            wordBreak="keep-all"
            whiteSpace="nowrap"
            width="full"
            overflowX="hidden"
            color={valueColor}
            userSelect="auto"
            _hover={{ overflow: "visible", cursor: "text" }}>
            {option.value ?? "-"}
          </Text>
        </ListItem>
      ))}
    </List>
  )
}
