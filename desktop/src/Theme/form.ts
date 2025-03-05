import { formAnatomy } from "@chakra-ui/anatomy"
import { createMultiStyleConfigHelpers } from "@chakra-ui/react"
import { mode } from "@chakra-ui/theme-tools"

const { definePartsStyle, defineMultiStyleConfig } = createMultiStyleConfigHelpers(formAnatomy.keys)
export const Form = defineMultiStyleConfig({
  baseStyle: definePartsStyle((props) => {
    return {
      helperText: {
        color: mode("gray.500", "gray.300")(props),
      },
    }
  }),
  variants: {
    contrast: definePartsStyle((props) => {
      return {
        helperText: {
          color: mode("gray.600", "gray.300")(props),
        },
      }
    }),
  },
})
