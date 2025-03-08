import { defineStyleConfig } from "@chakra-ui/react"
import { mode } from "@chakra-ui/theme-tools"

export const Text = defineStyleConfig({
  variants: {
    muted(props) {
      return {
        color: mode("gray.600", "gray.400")(props),
      }
    },
  },
})
