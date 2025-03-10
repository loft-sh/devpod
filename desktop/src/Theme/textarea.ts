import { defineStyleConfig } from "@chakra-ui/react"

export const Textarea = defineStyleConfig({
  variants: {
    outline: {
      borderColor: "gray.200",
      _dark: {
        borderColor: "gray.800",
      },
    },
  },
})
