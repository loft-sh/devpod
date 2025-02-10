import { Box, Checkbox, HStack, Heading, Icon, Text, VStack } from "@chakra-ui/react"
import dayjs from "dayjs"
import { ReactNode, useId } from "react"
import { HiClock, HiOutlineCode } from "react-icons/hi"
import { useNavigate } from "react-router"
import { IconTag } from "@/components"
import { Stack3D } from "@/icons"
import { Routes } from "@/routes"

type TWorkspaceCardHeaderProps = Readonly<{
  id: string
  statusBadge?: ReactNode
  controls?: ReactNode
  children?: ReactNode
  source?: ReactNode

  isSelected?: boolean
  onSelectionChange?: (isSelected: boolean) => void
}>
export function WorkspaceCardHeader({
  id,
  isSelected,
  onSelectionChange,
  statusBadge,
  controls,
  source,
  children,
}: TWorkspaceCardHeaderProps) {
  const checkboxID = useId()

  return (
    <>
      <VStack align="start" spacing={0}>
        <HStack w="full">
          {onSelectionChange && (
            <Checkbox
              id={checkboxID}
              paddingRight="2"
              isChecked={isSelected}
              onChange={(e) => onSelectionChange(e.target.checked)}
            />
          )}
          <Heading size="md">
            <HStack alignItems="baseline" justifyContent="space-between">
              <Text
                as="label"
                htmlFor={checkboxID}
                fontWeight="bold"
                maxWidth="23rem"
                overflow="hidden"
                whiteSpace="nowrap"
                textOverflow="ellipsis">
                {id}
              </Text>
              <Box transform="translateY(1px)">{statusBadge}</Box>
            </HStack>
          </Heading>
          <Box marginLeft="auto">{controls}</Box>
        </HStack>
        {source}
      </VStack>

      <HStack rowGap={2} marginTop={4} flexWrap="wrap" alignItems="center" paddingLeft="8">
        {children}
      </HStack>
    </>
  )
}

type TProviderProps = Readonly<{ name: string | undefined }>
function Provider({ name }: TProviderProps) {
  const navigate = useNavigate()

  return (
    <IconTag
      icon={<Stack3D />}
      label={name ?? "No provider"}
      info={name ? `Uses provider ${name}` : undefined}
      onClick={() => {
        if (!name) {
          return
        }

        navigate(Routes.toProvider(name))
      }}
    />
  )
}
WorkspaceCardHeader.Provider = Provider

type TIDEProps = Readonly<{ name: string }>
function IDE({ name }: TIDEProps) {
  return (
    <IconTag icon={<Icon as={HiOutlineCode} />} label={name} info={`Will be opened in ${name}`} />
  )
}
WorkspaceCardHeader.IDE = IDE

type TLastUsedProps = Readonly<{ timestamp: string }>
function LastUsed({ timestamp }: TLastUsedProps) {
  return (
    <IconTag
      icon={<Icon as={HiClock} />}
      label={dayjs(new Date(timestamp)).fromNow()}
      info={`Last used ${dayjs(new Date(timestamp)).fromNow()}`}
    />
  )
}
WorkspaceCardHeader.LastUsed = LastUsed
