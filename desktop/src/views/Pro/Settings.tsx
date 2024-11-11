import { HStack, Heading, VStack } from "@chakra-ui/react"
import { Settings as OSSSettings } from "../Settings"
import { BackToWorkspaces } from "./BackToWorkspaces"

export function Settings() {
  return (
    <VStack align="start">
      <BackToWorkspaces />
      <HStack align="center" justify="space-between" mb="6">
        <Heading fontWeight="thin">Settings</Heading>
      </HStack>
      <OSSSettings />
    </VStack>
  )
}
