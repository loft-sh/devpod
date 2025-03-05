import { inputAnatomy } from "@chakra-ui/anatomy"
import { createMultiStyleConfigHelpers } from "@chakra-ui/react"

const { definePartsStyle, defineMultiStyleConfig } = createMultiStyleConfigHelpers(
  inputAnatomy.keys
)
export const Input = defineMultiStyleConfig({
  baseStyle: definePartsStyle({
    addon: {
      borderColor: "gray.200",
      _dark: {
        borderColor: "gray.700",
      },
    },
    field: {
      borderColor: "gray.200",
      _dark: {
        borderColor: "gray.700",
      },
    },
  }),
})
