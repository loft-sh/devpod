import {
  Button,
  Card,
  Image,
  CardBody,
  CardFooter,
  CardHeader,
  Checkbox,
  Heading,
  Stack,
  Text,
  HStack,
  Box,
  IconButton,
  Popover,
  PopoverTrigger,
  PopoverContent,
  PopoverArrow,
  PopoverCloseButton,
  PopoverHeader,
  PopoverBody,
  VStack,
  Tooltip,
} from "@chakra-ui/react"
import { useWorkspace } from "../../contexts"
import { TWorkspaceID } from "../../types"
import { Pause, Trash } from "../../icons"
import { ArrowPath } from "../../icons/ArrowPath"
import CodeImage from "../../images/code.jpg"
import { Ellipsis } from "../../icons/Ellipsis"
import dayjs from "dayjs"

type TWorkspaceCardProps = Readonly<{
  workspaceID: TWorkspaceID
  onSelectionChange?: (isSelected: boolean) => void
}>

export function WorkspaceCard({ workspaceID, onSelectionChange }: TWorkspaceCardProps) {
  const workspace = useWorkspace(workspaceID)
  if (workspace.data === undefined) {
    return null
  }

  const { id, provider, status, ide } = workspace.data

  return (
    <Card
      key={id}
      direction={{ base: "row", sm: "row" }}
      width={"100%"}
      maxWidth={"600px"}
      overflow="hidden"
      variant="outline">
      <Image
        objectFit="cover"
        maxW={{ base: "100%", sm: "200px" }}
        src={CodeImage}
        alt="Project Image"
      />

      <Stack>
        <CardHeader display="flex" width="full" justifyContent="space-between">
          <Heading size="md">
            <HStack>
              <Text fontWeight="bold">{id}</Text>
              <Tooltip label={`Workspace is ${status}`}>
                <Box
                  as={"span"}
                  display={"inline-block"}
                  backgroundColor={status === "Running" ? "green" : "orange"}
                  borderRadius={"20px"}
                  width={"10px"}
                  height={"10px"}
                  position={"relative"}
                  top={"1px"}
                />
              </Tooltip>
            </HStack>
          </Heading>
          {onSelectionChange !== undefined && (
            <Checkbox onChange={(e) => onSelectionChange(e.target.checked)} />
          )}
        </CardHeader>
        <CardBody>
          {provider?.name && <Text>Provider: {provider.name}</Text>}
          <Text>Last Used: {dayjs(new Date(workspace.data.lastUsed)).fromNow()}</Text>
        </CardBody>
        <CardFooter>
          {workspace.data.status !== "Busy" && workspace.data.status !== "NotFound" && (
            <HStack spacing={"2"}>
              <Button
                colorScheme="primary"
                onClick={() => workspace.start({ id, ideConfig: ide })}
                isLoading={
                  workspace.current?.name === "start" && workspace.current.status === "pending"
                }>
                {workspace.data.status === "Stopped" ? "Start" : "Open"}
              </Button>
              {workspace.data.status === "Running" && (
                <Tooltip label={"Stop workspace"}>
                  <IconButton
                    aria-label="Stop workspace"
                    variant="ghost"
                    colorScheme="gray"
                    onClick={() => workspace.stop()}
                    icon={<Pause width={"16px"} />}
                    isLoading={
                      workspace.current?.name === "stop" && workspace.current.status === "pending"
                    }
                  />
                </Tooltip>
              )}
              <Tooltip label={"Delete workspace"}>
                <IconButton
                  aria-label="Delete workspace"
                  variant="ghost"
                  colorScheme="gray"
                  icon={<Trash width={"16px"} />}
                  onClick={() => workspace.remove()}
                  isLoading={
                    workspace.current?.name === "remove" && workspace.current.status === "pending"
                  }
                />
              </Tooltip>
              <Popover trigger={"hover"}>
                <PopoverTrigger>
                  <IconButton
                    aria-label="More actions"
                    variant="ghost"
                    colorScheme="gray"
                    icon={<Ellipsis width={"16px"} />}
                    onClick={() => {}}
                  />
                </PopoverTrigger>
                <PopoverContent>
                  <PopoverArrow />
                  <Box padding={"10px"}>
                    <VStack>
                      <Button>Open with...</Button>
                      <Button>Edit</Button>
                      <Button>Rebuild</Button>
                    </VStack>
                  </Box>
                </PopoverContent>
              </Popover>
            </HStack>
          )}
        </CardFooter>
      </Stack>
    </Card>
  )
}
