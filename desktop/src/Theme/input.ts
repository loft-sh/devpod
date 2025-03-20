import { inputAnatomy } from "@chakra-ui/anatomy"
import { createMultiStyleConfigHelpers } from "@chakra-ui/react"

const { definePartsStyle, defineMultiStyleConfig } = createMultiStyleConfigHelpers(
  inputAnatomy.keys
)
export const Input = defineMultiStyleConfig({
  variants: {
    outline: definePartsStyle(() => {
      return {
        addon: {
          borderColor: "gray.200",
          _dark: {
            borderColor: "gray.800",
          },
        },
        field: {
          borderColor: "gray.200",
          _dark: {
            borderColor: "gray.800",
          },
        },
      }
    }),
  },
})
