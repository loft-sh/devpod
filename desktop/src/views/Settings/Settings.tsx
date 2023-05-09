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
  Text,
  useColorModeValue,
  VStack,
} from "@chakra-ui/react"
import { ReactNode } from "react"
import { client } from "../../client"
import { ToolbarTitle, useInstallCLI } from "../../components"
import { TSettings, useChangeSettings } from "../../contexts"

export function Settings() {
  const { settings, set } = useChangeSettings()
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
