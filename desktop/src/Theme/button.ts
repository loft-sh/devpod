import { defineStyleConfig } from "@chakra-ui/react"
import { mode } from "@chakra-ui/theme-tools"

export const Button = defineStyleConfig({
  defaultProps: {
    size: "sm",
  },
  variants: {
    primary: {
      color: "white",
      borderColor: "primary.600",
      borderWidth: 1,
      backgroundColor: "primary.500",
      _hover: {
        backgroundColor: "primary.600",
        _disabled: {
          background: "primary.500",
        },
      },
    },
    ["solid-outline"](props) {
      return {
        color: mode("gray.600", "gray.400")(props),
        borderColor: mode("gray.300", "whiteAlpha.300")(props),
        borderWidth: 1,
        ".chakra-button__group[data-attached][data-orientation=horizontal] > &:not(:last-of-type)":
          { marginEnd: "-1px" },
        ".chakra-button__group[data-attached][data-orientation=vertical] > &:not(:last-of-type)": {
          marginBottom: "-1px",
        },
        backgroundColor: mode("gray.100", "whiteAlpha.200")(props),
        _hover: {
          backgroundColor: mode("gray.200", "whiteAlpha.300")(props),
        },
        _active: {
          backgroundColor: mode("gray.300", "whiteAlpha.400")(props),
        },
      }
    },
    announcement({ theme }) {
      const from = theme.colors.primary["900"]
      const to = theme.colors.primary["600"]

      return {
        color: "white",
        transition: "background 150ms",
        fontWeight: "regular",
        background: `linear-gradient(170deg, ${from} 15%, ${to})`,
        backgroundSize: "130% 130%",
        _hover: {
          backgroundPosition: "90% 50%",
        },
        _active: {
          boxShadow: "inset 0 0 3px 2px rgba(0, 0, 0, 0.2)",
        },
      }
    },
    proWorkspaceIDE(props) {
      return {
        color: mode("white", "black"),
        fontWeight: "semibold",
        backgroundColor: mode("gray.600", "whiteAlpha.200")(props),
        _hover: {
          backgroundColor: mode("gray.700", "whiteAlpha.300")(props),
        },
        _active: {
          backgroundColor: mode("gray.800", "whiteAlpha.400")(props),
        },
      }
    },
  },
})
