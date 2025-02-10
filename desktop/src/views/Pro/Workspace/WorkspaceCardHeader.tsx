import { IDEIcon } from "@/components"
import { ProWorkspaceInstance, useSettings } from "@/contexts"
import {
  ArrowCycle,
  ArrowPath,
  Cog,
  Ellipsis,
  GitBranch,
  GitCommit,
  GitPR,
  GitSubPath,
  Pause,
  Trash,
} from "@/icons"
import { getDisplayName, getIDEDisplayName } from "@/lib"
import { TIDE, TIDEs, TWorkspaceSource } from "@/types"
import { useIDEs } from "@/useIDEs"
import { ChevronDownIcon } from "@chakra-ui/icons"
import {
  Button,
  ButtonGroup,
  HStack,
  Heading,
  IconButton,
  Menu,
  MenuButton,
  MenuItem,
  MenuList,
  Portal,
  Text,
  TextProps,
  VStack,
} from "@chakra-ui/react"
import { ReactNode, createContext, useContext } from "react"
type TWorkspaceCardHeaderContext = ProWorkspaceInstance
const WorkspaceCardHeaderContext = createContext<TWorkspaceCardHeaderContext>(null!)

type TWorkspaceCardHeaderProps = Readonly<{
  instance: ProWorkspaceInstance
  children?: ReactNode
  showSource?: boolean
}>
export function WorkspaceCardHeader({
  instance,
  children,
  showSource = true,
}: TWorkspaceCardHeaderProps) {
  const source = instance.status?.source
  const sourceDetail = getSourceDetail(source)

  return (
    <HStack justify="space-between" align="start">
      <VStack align="start" spacing={0}>
        {showSource && (
          <Text color="gray.500">
            {source?.gitRepository || source?.image || source?.localFolder}
          </Text>
        )}
        <Heading size="md" my="1">
          <Text
            fontWeight="bold"
            maxW="50rem"
            overflow="hidden"
            whiteSpace="nowrap"
            textOverflow="ellipsis">
            {getDisplayName(instance, instance.id)}
          </Text>
        </Heading>
        {showSource && sourceDetail ? sourceDetail : null}
      </VStack>
      <WorkspaceCardHeaderContext.Provider value={instance}>
        {children}
      </WorkspaceCardHeaderContext.Provider>
    </HStack>
  )
}

type TControlsProps = Readonly<{
  onOpenClicked: (ideName: string) => void
  onStopClicked: VoidFunction
  onResetClicked: VoidFunction
  onRebuildClicked: VoidFunction
  onDeleteClicked: VoidFunction
  onTroubleshootClicked: VoidFunction
}>
export function Controls({
  onOpenClicked,
  onStopClicked,
  onResetClicked,
  onRebuildClicked,
  onDeleteClicked,
  onTroubleshootClicked,
}: TControlsProps) {
  const { ides, defaultIDE } = useIDEs()
  const settings = useSettings()
  const instance = useContext(WorkspaceCardHeaderContext)
  const ide = getWorkspaceIDE(instance, ides, defaultIDE, settings.fixedIDE)

  return (
    <ButtonGroup size="sm" onClick={(e) => e.stopPropagation()}>
      <ButtonGroup variant="proWorkspaceIDE" isAttached onClick={(e) => e.stopPropagation()}>
        {ide && (
          <Button onClick={() => onOpenClicked(ide.name!)}>
            <HStack>
              {ide.name !== "none" && <IDEIcon ide={ide} width={6} height={6} size="sm" />}
              <Text>{ide.name === "none" ? "SSH" : getIDEDisplayName(ide)}</Text>
            </HStack>
          </Button>
        )}
        <Menu>
          <MenuButton
            as={IconButton}
            borderLeftColor="white"
            borderLeftStyle="solid"
            borderLeftWidth="thin"
            aria-label="Show more IDEs"
            icon={<ChevronDownIcon boxSize="5" />}
          />
          <MenuList>
            {ides?.map((ide) => (
              <MenuItem
                onClick={() => onOpenClicked(ide.name!)}
                key={ide.name}
                value={ide.name!}
                icon={<IDEIcon ide={ide} width={6} height={6} size="sm" />}>
                {getIDEDisplayName(ide)}
              </MenuItem>
            ))}
          </MenuList>
        </Menu>
      </ButtonGroup>

      <Menu placement="bottom">
        <MenuButton
          as={IconButton}
          colorScheme="gray"
          variant="ghost"
          aria-label="More actions"
          _hover={{ bgColor: "gray.200" }}
          _active={{ bgColor: "gray.300" }}
          icon={<Ellipsis transform={"rotate(90deg)"} boxSize={5} />}
        />
        <Portal>
          <MenuList mr="4">
            <MenuItem isDisabled={false} onClick={onStopClicked} icon={<Pause boxSize={4} />}>
              Stop...
            </MenuItem>
            <MenuItem
              icon={<ArrowPath boxSize={4} />}
              onClick={onRebuildClicked}
              isDisabled={false}>
              Rebuild...
            </MenuItem>
            <MenuItem icon={<ArrowCycle boxSize={4} />} onClick={onResetClicked} isDisabled={false}>
              Reset...
            </MenuItem>
            <MenuItem
              fontWeight="normal"
              icon={<Cog boxSize={4} />}
              onClick={onTroubleshootClicked}>
              Troubleshoot
            </MenuItem>
            <MenuItem
              isDisabled={false}
              fontWeight="normal"
              icon={<Trash boxSize={4} />}
              onClick={onDeleteClicked}>
              Delete...
            </MenuItem>
          </MenuList>
        </Portal>
      </Menu>
    </ButtonGroup>
  )
}
WorkspaceCardHeader.Controls = Controls

function getSourceDetail(source: TWorkspaceSource | undefined): ReactNode | undefined {
  if (!source) {
    return undefined
  }

  const sharedProps: TextProps = { color: "gray.500", gap: "1", align: "center" }

  if (source.gitBranch) {
    return (
      <HStack {...sharedProps}>
        <GitBranch boxSize="4" />
        <Text>{source.gitBranch}</Text>
      </HStack>
    )
  }

  if (source.gitCommit) {
    return (
      <HStack {...sharedProps}>
        <GitCommit boxSize="4" />
        <Text>{source.gitCommit}</Text>
      </HStack>
    )
  }

  if (source.gitPRReference) {
    return (
      <HStack {...sharedProps}>
        <GitPR boxSize="4" />
        <Text>{source.gitPRReference}</Text>
      </HStack>
    )
  }

  if (source.gitSubPath) {
    return (
      <HStack {...sharedProps}>
        <GitSubPath boxSize="4" />
        <Text>{source.gitSubPath}</Text>
      </HStack>
    )
  }
}

function getWorkspaceIDE(
  instance: ProWorkspaceInstance,
  ides: TIDEs | undefined,
  defaultIDE: TIDE | undefined,
  fixedIDE: boolean
): TIDE | undefined {
  if (fixedIDE && defaultIDE) {
    return defaultIDE
  }
  const instanceIDEName = instance.status?.ide?.name
  const instanceIDE = ides?.find((ide) => ide.name === instanceIDEName)

  if (instanceIDE) {
    return instanceIDE
  }

  if (defaultIDE) {
    return defaultIDE
  }

  return ides?.find((ide) => ide.name === "vscode")
}
