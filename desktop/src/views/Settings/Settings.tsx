import {
  Checkbox,
  Code,
  Heading,
  HStack,
  Radio,
  RadioGroup,
  Text,
  useColorModeValue,
  VStack,
} from "@chakra-ui/react"
import { ReactNode } from "react"
import { ToolbarTitle } from "../../components"
import { TSettings, useChangeSettings } from "../../contexts"

export function Settings() {
  const { settings, set } = useChangeSettings()

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
