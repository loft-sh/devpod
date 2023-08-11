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
    ["primary-outline"](props) {
      const hover = props.theme.colors.primary["500"]
      const active = props.theme.colors.primary["800"]

      return {
        color: mode("primary.600", "primary.400")(props),
        borderColor: mode("primary.600", "primary.400")(props),
        borderWidth: 1,
        backgroundColor: "transparent",
        _hover: {
          backgroundColor: `${hover}33`,
        },
        _active: {
          backgroundColor: `${active}33`,
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
  },
})
