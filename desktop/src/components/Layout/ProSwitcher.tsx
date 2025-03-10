import { client } from "@/client"
import { useProInstances, useProviders, useSettings } from "@/contexts"
import { CheckCircle, CircleWithArrow, DevPodProBadge, ExclamationTriangle } from "@/icons"
import {
  exists,
  canHealthCheck as isNewProProvider,
  useLoginProModal,
  useReLoginProModal,
} from "@/lib"
import { Routes } from "@/routes"
import { TProID, TProInstance, TProInstances, TProviderConfig } from "@/types"
import { useDeleteProviderModal } from "@/views/Providers/useDeleteProviderModal"
import { ChevronDownIcon, CloseIcon } from "@chakra-ui/icons"
import {
  Box,
  Button,
  ButtonGroup,
  HStack,
  Heading,
  Icon,
  IconButton,
  Link,
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
  useColorModeValue,
} from "@chakra-ui/react"
import dayjs from "dayjs"
import { Dispatch, ReactElement, SetStateAction, useEffect, useMemo, useState } from "react"
import { HiArrowRightOnRectangle, HiClock } from "react-icons/hi2"
import { useNavigate } from "react-router-dom"
import { IconTag } from "../Tag"

type TProInstanceWithProvider = TProInstance & Readonly<{ providerConfig: TProviderConfig | null }>
export function ProSwitcher() {
  const [[proInstances]] = useProInstances()
  const { modal: loginProModal, handleOpenLogin: handleConnectClicked } = useLoginProModal()
  const { modal: reLoginProModal, handleOpenLogin: handleReLoginClicked } = useReLoginProModal()
  const [isDeleting, setIsDeleting] = useState(false)

  const backgroundColor = useColorModeValue("white", "gray.900")
  const handleAnnouncementClicked = () => {
    client.open("https://devpod.sh/pro")
  }
  const { experimental_devPodPro } = useSettings()
  const isProUnauthenticated = proInstances?.some(({ authenticated }) => !authenticated)
  if (!experimental_devPodPro) {
    return (
      <Button
        variant="outline"
        leftIcon={<DevPodProBadge width="9" height="8" />}
        onClick={handleAnnouncementClicked}>
        Try DevPod Pro
      </Button>
    )
  }

  return (
    <>
      <Popover isLazy isOpen={isDeleting ? true : undefined}>
        <PopoverTrigger>
          <Button
            variant="outline"
            rightIcon={<ChevronDownIcon boxSize={6} />}
            {...(isProUnauthenticated && {
              leftIcon: <ExclamationTriangle boxSize={4} color="orange.300" />,
            })}>
            DevPod Pro
          </Button>
        </PopoverTrigger>
        <Portal>
          <PopoverContent backgroundColor={backgroundColor} zIndex="popover">
            <PopoverArrow backgroundColor={backgroundColor} />
            <ProPopoverContent
              proInstances={proInstances}
              onConnect={handleConnectClicked}
              setIsDeleting={setIsDeleting}
              onReLogin={(host) => handleReLoginClicked({ host })}
              emptyProInstances={<EmptyProInstances onConnect={handleConnectClicked} />}
            />
          </PopoverContent>
        </Portal>
      </Popover>
      {loginProModal}
      {reLoginProModal}
    </>
  )
}

type TProPopoverContentProps = Readonly<{
  proInstances: TProInstances | undefined
  emptyProInstances: ReactElement
  setIsDeleting: Dispatch<SetStateAction<boolean>>
  onConnect: VoidFunction
  onReLogin: (host: string) => void
}>
function ProPopoverContent({
  proInstances,
  emptyProInstances,
  setIsDeleting,
  onConnect,
  onReLogin,
}: TProPopoverContentProps) {
  const navigate = useNavigate()
  const hoverBgColor = useColorModeValue("gray.100", "gray.700")
  const [[providers]] = useProviders()
  const { newProInstances, legacyProInstances } = useMemo(() => {
    return (
      proInstances
        ?.map((instance) => {
          if (!instance.provider) {
            return { ...instance, providerConfig: null }
          }
          const providerConfig = providers?.[instance.provider]?.config
          if (!providerConfig) {
            return { ...instance, providerConfig: null }
          }

          return { ...instance, providerConfig }
        })
        .reduce(
          (acc, curr) => {
            if (!curr.providerConfig) {
              acc.legacyProInstances.push(curr)

              return acc
            }
            if (!isNewProProvider(curr.providerConfig)) {
              acc.legacyProInstances.push(curr)

              return acc
            }

            acc.newProInstances.push(curr)

            return acc
          },
          {
            newProInstances: [] as TProInstanceWithProvider[],
            legacyProInstances: [] as TProInstanceWithProvider[],
          }
        ) ?? {
        newProInstances: [] as TProInstanceWithProvider[],
        legacyProInstances: [] as TProInstanceWithProvider[],
      }
    )
  }, [proInstances, providers])

  return (
    <>
      <PopoverHeader>
        <VStack align="start" spacing="0">
          <Heading size="sm" as="h3">
            Your Pro Instances
          </Heading>
          <Text fontSize="xs">Manage DevPod Pro</Text>
        </VStack>
        <ButtonGroup variant="outline">
          <Tooltip label="Connect to Pro instance">
            <IconButton
              aria-label="Connect to Pro Instace"
              onClick={onConnect}
              icon={<Icon as={HiArrowRightOnRectangle} boxSize={5} />}
            />
          </Tooltip>
        </ButtonGroup>
      </PopoverHeader>
      <PopoverBody>
        <Box width="full" overflowY="auto" maxHeight="17rem" height="full" px="2">
          {proInstances === undefined || (proInstances.length === 0 && emptyProInstances)}
          {legacyProInstances.map((proInstance) => {
            const host = proInstance.host
            if (!host) {
              return null
            }

            return (
              <ProInstanceRow
                key={host}
                {...proInstance}
                host={host}
                onIsDeletingChanged={setIsDeleting}
                onLoginClicked={() => onReLogin(host)}
              />
            )
          })}
        </Box>

        <List>
          {newProInstances.map(({ host, authenticated }) => {
            if (!host) {
              return null
            }

            return (
              <ListItem key={host}>
                <Button
                  _hover={{ bg: hoverBgColor }}
                  variant="unstyled"
                  w="full"
                  px="4"
                  h="12"
                  onClick={() => navigate(Routes.toProInstance(host))}>
                  <HStack w="full" justify="space-between">
                    <Text maxW="50%" overflow="hidden" textOverflow="ellipsis">
                      {host}
                    </Text>
                    <HStack>
                      {authenticated != null && (
                        <Box
                          boxSize="2"
                          bg={authenticated ? "green.400" : "orange.400"}
                          rounded="full"
                        />
                      )}
                      <Text fontSize="xs" fontWeight="normal">
                        {host}
                      </Text>
                      <CircleWithArrow boxSize={5} />
                    </HStack>
                  </HStack>
                </Button>
              </ListItem>
            )
          })}
        </List>
      </PopoverBody>
    </>
  )
}

type TEmptyProInstancesProps = Readonly<{
  onConnect: VoidFunction
}>
function EmptyProInstances({ onConnect }: TEmptyProInstancesProps) {
  return (
    <VStack align="start" padding="2" spacing="0">
      <Text fontWeight="bold">No Pro instances</Text>
      <Text lineHeight={"1.2rem"} fontSize="sm" color="gray.500">
        You don&apos;t have any Pro instances set up. Connect to an existing Instance or create a
        new one. <br />
        <Link color="primary.600" onClick={() => client.open("https://devpod.sh/pro")}>
          Learn more
        </Link>
      </Text>
      <Button marginTop="4" variant="primary" onClick={onConnect}>
        Login to Pro
      </Button>
    </VStack>
  )
}

type TProInstaceRowProps = Omit<TProInstance, "host"> &
  Readonly<{
    host: TProID
    onIsDeletingChanged: (isDeleting: boolean) => void
    onLoginClicked?: VoidFunction
  }>
function ProInstanceRow({
  host,
  creationTimestamp,
  onIsDeletingChanged,
  authenticated,
  onLoginClicked,
}: TProInstaceRowProps) {
  const [, { disconnect }] = useProInstances()
  const {
    modal: deleteProviderModal,
    open: openDeleteProviderModal,
    isOpen,
  } = useDeleteProviderModal(host, "Pro instance", "disconnect", () => disconnect.run({ id: host }))
  useEffect(() => {
    onIsDeletingChanged(isOpen)
    // `onIsDeletingChanged` is expected to be a stable reference
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen])

  return (
    <>
      <HStack width="full" padding="2" justifyContent="space-between">
        <VStack align="start" spacing="0" fontSize="sm">
          <HStack>
            <Text fontWeight="bold">{host}</Text>
            {exists(authenticated) && (
              <IconTag
                variant="ghost"
                icon={
                  authenticated ? (
                    <CheckCircle color={"green.300"} />
                  ) : (
                    <ExclamationTriangle color="orange.300" />
                  )
                }
                label=""
                paddingInlineStart="0"
                info={authenticated ? "Authenticated" : "Not Authenticated"}
                {...(authenticated ? {} : { onClick: onLoginClicked, cursor: "pointer" })}
              />
            )}
          </HStack>
          <HStack>
            {exists(creationTimestamp) && (
              <IconTag
                paddingInlineStart="0"
                variant="ghost"
                icon={<Icon as={HiClock} />}
                label={dayjs(new Date(creationTimestamp)).format("MMM D, YY")}
                info={`Created ${dayjs(new Date(creationTimestamp)).fromNow()}`}
              />
            )}
          </HStack>
        </VStack>

        {exists(host) && (
          <Tooltip label="Disconnect from Instance">
            <IconButton
              variant="ghost"
              size="xs"
              aria-label="Disconnect from Instance"
              onClick={openDeleteProviderModal}
              icon={<CloseIcon />}
            />
          </Tooltip>
        )}
      </HStack>

      {deleteProviderModal}
    </>
  )
}
