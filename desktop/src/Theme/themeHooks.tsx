import { useColorModeValue } from "@chakra-ui/react"

export function useBorderColor() {
  return useColorModeValue("gray.200", "gray.700")
}
