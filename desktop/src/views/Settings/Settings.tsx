import {
  Button,
  Checkbox,
  Code,
  Divider,
  Heading,
  HeadingProps,
  HStack,
  Icon,
  Radio,
  RadioGroup,
  Select,
  Text,
  useColorModeValue,
  VStack,
} from "@chakra-ui/react"
import { ReactNode } from "react"
import { HiMagnifyingGlassPlus } from "react-icons/hi2"
import { client } from "../../client"
import { ToolbarTitle, useInstallCLI } from "../../components"
import { TSettings, useChangeSettings } from "../../contexts"
import { getIDEDisplayName } from "../../lib"
import { useWelcomeModal } from "../../useWelcomeModal"
import { useAgentURLOption } from "./useContextOptions"
import { useIDESettings } from "./useIDESettings"

export function Settings() {
  const { settings, set } = useChangeSettings()
  const { ides, defaultIDE, updateDefaultIDE } = useIDESettings()
  const { input: agentURLInput, helpText: agentURLHelpText } = useAgentURLOption()
  const { modal: welcomeModal, show: showWelcomeModal } = useWelcomeModal()
  const {
    badge: installCLIBadge,
    button: installCLIButton,
    helpText: installCLIHelpText,
    errorMessage: installCLIErrorMessage,
  } = useInstallCLI()
  const headingProps: HeadingProps = { marginBottom: 2, as: "h4", size: "md" }

  return (
    <>
      <ToolbarTitle>
        <Heading as="h3" size="sm">
          Settings
        </Heading>
      </ToolbarTitle>
      <VStack align="start" spacing={6} paddingBottom={8}>
        <VStack align="start">
          <Heading {...headingProps}>General</Heading>
          <RadioGroup
            value={settings.sidebarPosition}
            onChange={(newValue: TSettings["sidebarPosition"]) => set("sidebarPosition", newValue)}>
            <HStack>
              <Radio value="left">Left</Radio>
              <Radio value="right">Right</Radio>
            </HStack>
          </RadioGroup>
          <SettingDescription>Position the sidebar</SettingDescription>

          <VStack align="start" paddingTop="2">
            <Button variant="outline" onClick={() => showWelcomeModal({ cancellable: true })}>
              Show Intro
            </Button>
            <SettingDescription>Show the introduction to DevPod again</SettingDescription>
          </VStack>

          <VStack align="start" paddingTop="2">
            <HStack>
              <Select
                onChange={(e) => set("zoom", e.target.value as TSettings["zoom"])}
                value={settings.zoom}>
                <option value={"sm"}>Small</option>
                <option value={"md"}>Regular</option>
                <option value={"lg"}>Large</option>
                <option value={"xl"}>Extra Large</option>
              </Select>
              <Icon as={HiMagnifyingGlassPlus} boxSize="6" color="gray.600" />
            </HStack>
            <SettingDescription>Adjust the zoom level</SettingDescription>
          </VStack>

          <VStack align="start" paddingTop="2">
            {agentURLInput}
            <SettingDescription>{agentURLHelpText}</SettingDescription>
          </VStack>
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
            {ides?.map((ide) => (
              <option key={ide.name} value={ide.name!}>
                {getIDEDisplayName(ide)}
              </option>
            ))}
          </Select>
          <SettingDescription>
            Select the default IDE you&apos;re using for workspaces. This will be overridden
            whenever you create a workspace with a different IDE
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

        <VStack align="start">
          <Heading {...headingProps}>Experimental</Heading>

          <Checkbox
            isChecked={settings.experimental_multiDevcontainer}
            onChange={(e) => set("experimental_multiDevcontainer", e.target.checked)}>
            Check workspaces for multiple devcontainers
          </Checkbox>
          <SettingDescription>
            Whenever new workspaces are created, check if there are multiple devcontainers in the
            source. This might take a while for larger repositories.
          </SettingDescription>

          <Checkbox
            isChecked={settings.experimental_fleet}
            onChange={(e) => set("experimental_fleet", e.target.checked)}>
            JetBrains Fleet
          </Checkbox>
          <Checkbox
            isChecked={settings.experimental_jupyterNotebooks}
            onChange={(e) => set("experimental_jupyterNotebooks", e.target.checked)}>
            Jupyter Notebooks
          </Checkbox>
        </VStack>

        <Divider />

        <Heading {...headingProps} color="red.600">
          Danger Zone
        </Heading>
        <Button variant="outline" colorScheme="red" onClick={() => client.quit()}>
          Quit DevPod
        </Button>
      </VStack>

      {welcomeModal}
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
