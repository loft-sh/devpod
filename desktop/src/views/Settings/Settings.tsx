import {
  Button,
  Checkbox,
  Code,
  Divider,
  Heading,
  HeadingProps,
  HStack,
  Radio,
  RadioGroup,
  Select,
  Text,
  useColorModeValue,
  VStack,
} from "@chakra-ui/react"
import { ReactNode } from "react"
import { client } from "../../client"
import { ToolbarTitle, useInstallCLI } from "../../components"
import { TSettings, useChangeSettings } from "../../contexts"
import { QueryKeys } from "../../queryKeys"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { getIDEDisplayName } from "../../lib"
import { TIDE } from "../../types"

export function Settings() {
  const queryClient = useQueryClient()
  const { settings, set } = useChangeSettings()
  const {
    badge: installCLIBadge,
    button: installCLIButton,
    helpText: installCLIHelpText,
    errorMessage: installCLIErrorMessage,
  } = useInstallCLI()
  const idesQuery = useQuery({
    queryKey: QueryKeys.IDES,
    queryFn: async () => (await client.ides.listAll()).unwrap(),
  })
  const { mutate: updateDefaultIDE } = useMutation({
    mutationFn: async ({ ide }: { ide: NonNullable<TIDE["name"]> }) => {
      ;(await client.ides.useIDE(ide)).unwrap()
    },
    onSettled: () => {
      queryClient.invalidateQueries(QueryKeys.IDES)
    },
  })
  const defaultIDE = idesQuery.data?.find((ide) => ide.default)
  const headingProps: HeadingProps = { marginBottom: 2, as: "h4", size: "md" }

  return (
    <>
      <ToolbarTitle>
        <Heading as="h3" size="sm">
          Settings
        </Heading>
      </ToolbarTitle>
      <VStack align="start" spacing={6}>
        <VStack align="start">
          <Heading {...headingProps}>Appearance</Heading>

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
          <Heading {...headingProps}>Debugging</Heading>

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
          <Heading {...headingProps}>IDE</Heading>

          <Select
            maxWidth="52"
            textTransform="capitalize"
            onChange={(e) => updateDefaultIDE({ ide: e.target.value })}
            value={defaultIDE ? defaultIDE.name! : undefined}>
            {idesQuery.data?.map((ide) => (
              <option key={ide.name} value={ide.name!}>
                {getIDEDisplayName(ide)}
              </option>
            ))}
          </Select>
          <SettingDescription>
            Select the default IDE you&apos;re using for workspaces. This will be overriden whenever
            you create a workspace with a different IDE
          </SettingDescription>

          <Checkbox
            isChecked={settings.fixedIDE}
            onChange={(e) => set("fixedIDE", e.target.checked)}>
            Always use this IDE
          </Checkbox>
          <SettingDescription>
            Open workspaces with the selected IDE by default. Prevents the app from storing the last
            IDE you used
          </SettingDescription>
        </VStack>

        <VStack align="start">
          <HStack marginBottom={headingProps.marginBottom}>
            <Heading as={headingProps.as} size={headingProps.size}>
              CLI
            </Heading>
            {installCLIBadge}
          </HStack>

          {installCLIButton}
          <SettingDescription>{installCLIHelpText}</SettingDescription>
          {installCLIErrorMessage}
        </VStack>
        <Divider />
        <Heading {...headingProps} color="red.600">
          Danger Zone
        </Heading>
        <Button variant="outline" colorScheme="red" onClick={() => client.quit()}>
          Quit DevPod
        </Button>
      </VStack>
    </>
  )
}
type TSettingDescriptionProps = Readonly<{ children: ReactNode }>
function SettingDescription({ children }: TSettingDescriptionProps) {
  const descriptionColor = useColorModeValue("gray.500", "gray.400")

  return (
    <Text paddingLeft={6} color={descriptionColor} fontSize="sm">
      {children}
    </Text>
  )
}
