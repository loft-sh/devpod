import { HStack, Heading, VStack } from "@chakra-ui/react"
import { BackToWorkspaces } from "./BackToWorkspaces"

export function Profile() {
  return (
    <VStack align="start">
      <BackToWorkspaces />
      <HStack align="center" justify="space-between" mb="6">
        <Heading fontWeight="thin">Profile</Heading>
      </HStack>
    </VStack>
  )
}
