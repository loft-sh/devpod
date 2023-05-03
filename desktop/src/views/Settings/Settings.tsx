import {
  Button,
  Checkbox,
  Code,
  Heading,
  HStack,
  Radio,
  RadioGroup,
  Text,
  Tooltip,
  useColorModeValue,
  VStack,
} from "@chakra-ui/react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { ReactNode } from "react"
import { client } from "../../client"
import { ErrorMessageBox, ToolbarTitle } from "../../components"
import { TSettings, useChangeSettings } from "../../contexts"
import { CheckCircle, ExclamationCircle } from "../../icons"
import { isError } from "../../lib"
import { QueryKeys } from "../../queryKeys"

export function Settings() {
  const { settings, set } = useChangeSettings()
  const { data: isCLIInstalled } = useQuery<boolean>({
    queryKey: QueryKeys.IS_CLI_INSTALLED,
    queryFn: async () => {
      return (await client.isCLIInstalled()).unwrap()!
    },
  })
  const queryClient = useQueryClient()
  const {
    mutate: addBinaryToPath,
    isLoading,
    error,
    status,
  } = useMutation({
    mutationFn: async () => {
      ;(await client.installCLI()).unwrap()
    },
    onSettled: () => {
      queryClient.invalidateQueries(QueryKeys.IS_CLI_INSTALLED)
    },
  })

  return (
    <>
      <ToolbarTitle>
        <Heading as="h3" size="sm">
          Settings
        </Heading>
      </ToolbarTitle>
      <VStack align="start" spacing={10}>
        <VStack align="start">
          <Heading as="h4" size="md" marginBottom={4}>
            Appearance
          </Heading>

          <RadioGroup
            value={settings.sidebarPosition}
            onChange={(newValue: TSettings["sidebarPosition"]) => set("sidebarPosition", newValue)}>
            <HStack>
              <Radio value="left">Left</Radio>
              <Radio value="right">Right</Radio>
            </HStack>
          </RadioGroup>
          <SettingDescription>Position the primary sidebar</SettingDescription>
        </VStack>

        <VStack align="start">
          <Heading as="h4" size="md" marginBottom={4}>
            Debugging
          </Heading>

          <Checkbox
            isChecked={settings.debugFlag}
            onChange={(e) => set("debugFlag", e.target.checked)}>
            Use --debug
          </Checkbox>
          <SettingDescription>
            Run all devpods command with the <Code>--debug</Code> flag, making it easier to
            troubleshoot
          </SettingDescription>
        </VStack>

        <VStack align="start">
          <HStack marginBottom={4}>
            <Heading as="h4" size="md">
              CLI
            </Heading>
            {isCLIInstalled ? (
              <Tooltip label="Installed">
                <CheckCircle boxSize={5} color="green.500" />
              </Tooltip>
            ) : (
              <Tooltip label="Not Installed">
                <ExclamationCircle boxSize={5} color="red.500" />
              </Tooltip>
            )}
          </HStack>

          <Button
            isLoading={isLoading}
            onClick={() => addBinaryToPath()}
            isDisabled={status === "success"}>
            Add CLI to Path
          </Button>
          <SettingDescription>
            Adds the DevPod CLI to your local users <Code>$PATH</Code>
          </SettingDescription>
          {isError(error) && <ErrorMessageBox error={error} />}
        </VStack>
      </VStack>
    </>
  )
}
type TSettingDescriptionProps = Readonly<{ children: ReactNode }>
function SettingDescription({ children }: TSettingDescriptionProps) {
  const descriptionColor = useColorModeValue("gray.500", "gray.400")

  return (
    <Text paddingLeft={6} color={descriptionColor}>
      {children}
    </Text>
  )
}
