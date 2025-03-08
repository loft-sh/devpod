import {
  ConfigureProviderOptionsForm,
  TOptionWithID,
  useProviderDisplayOptions,
} from "@/views/Providers"
import { mergeOptionDefinitions } from "@/views/Providers/helpers"
import { ViewIcon } from "@chakra-ui/icons"
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
  VStack,
  useDisclosure,
} from "@chakra-ui/react"
import { ReactElement, useCallback, useRef, useState } from "react"
import { HiPencil } from "react-icons/hi2"
import { TNamedProvider } from "../../../types"

type TProviderOptionsPopoverProps = Readonly<{ provider: TNamedProvider; trigger: ReactElement }>
export function ProviderOptionsPopover({ provider, trigger }: TProviderOptionsPopoverProps) {
  const { isOpen, onClose, onOpen } = useDisclosure()
  const bodyRef = useRef<HTMLDivElement | null>(null)

  const options = useProviderDisplayOptions(
    mergeOptionDefinitions(provider.state?.options ?? {}, provider.config?.options ?? {}),
    provider.config?.optionGroups ?? []
  )
  const [isLocked, setIsLocked] = useState(false)
  const [isEditing, setIsEditing] = useState(false)
  const reset = () => {
    setIsEditing(false)
    setIsLocked(false)
    onClose()
  }
  const handleEditClicked = () => {
    setIsEditing((x) => !x)
    setIsLocked(true)
  }

  const contentRef = useRef<HTMLElement | null>(null)
  const contentCallbackRef = useCallback((ref: HTMLElement | null) => {
    const handlerOpts = { capture: true }
    const handler = (event: KeyboardEvent) => {
      event.stopPropagation()
      event.preventDefault()
      const isCurrentTarget = event.target === contentRef.current
      if (isCurrentTarget && event.code === "KeyE") {
        setIsEditing((x) => !x)
        setIsLocked(true)
      }
    }
    if (ref === null) {
      contentRef.current?.removeEventListener("keyup", handler, handlerOpts)
      contentRef.current = ref

      return
    }

    contentRef.current = ref
    contentRef.current.addEventListener("keyup", handler, handlerOpts)
  }, [])

  const handleContentAnimationEnd = () => {
    if (isOpen) {
      contentRef.current?.focus()
    }
  }

  return (
    <Popover
      isOpen={isOpen}
      onClose={reset}
      trigger={isLocked ? "click" : "hover"}
      placement="top"
      isLazy
      onOpen={onOpen}>
      <PopoverTrigger>{trigger}</PopoverTrigger>
      <Portal>
        <PopoverContent
          onAnimationEnd={handleContentAnimationEnd}
          ref={contentCallbackRef}
          minWidth={isEditing ? "2xl" : "sm"}
          height="full">
          <PopoverArrow />
          <PopoverHeader fontWeight="semibold">
            <VStack align="start" spacing="0">
              <Heading size="sm" as="h3">
                {provider.name}
              </Heading>
              <Text fontSize="xs">Current provider configuration</Text>
            </VStack>
            <ButtonGroup variant="outline">
              <Tooltip label={!isEditing ? "Edit Provider (e)" : "View Configuration (e)"}>
                <IconButton
                  aria-label={!isEditing ? "Edit Provider" : "View Configuration"}
                  onClick={handleEditClicked}
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
  return (
    <List width="full" marginBottom="2">
      {options.map((option, i) => {
        let value = option.value ?? "-"
        if (option.password) {
          value = "*".repeat(value.length)
        }

        return (
          <ListItem key={i} width="full" display="flex" flexFlow="row nowrap">
            <Tooltip label={option.displayName}>
              <Text
                whiteSpace="nowrap"
                wordBreak="keep-all"
                marginRight="4"
                width="40"
                minWidth="40"
                overflowX="hidden"
                textOverflow="ellipsis">
                {option.displayName}
              </Text>
            </Tooltip>
            {option.enum ? (
              <Text
                textOverflow="ellipsis"
                wordBreak="keep-all"
                whiteSpace="nowrap"
                width="full"
                overflowX="hidden"
                variant="muted"
                userSelect="text"
                _hover={{ overflow: "visible", cursor: "text" }}>
                {option.enum.find((e) => e.value === value)?.displayName ?? value}
              </Text>
            ) : (
              <Text
                textOverflow="ellipsis"
                wordBreak="keep-all"
                whiteSpace="nowrap"
                width="full"
                overflowX="hidden"
                variant="muted"
                userSelect="text"
                _hover={{ overflow: "visible", cursor: "text" }}>
                {value}
              </Text>
            )}
          </ListItem>
        )
      })}
    </List>
  )
}
