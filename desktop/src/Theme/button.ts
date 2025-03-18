import { defineStyleConfig } from "@chakra-ui/react"
import { mode } from "@chakra-ui/theme-tools"
import { theme as defaultTheme } from "@chakra-ui/theme"

export const Button = defineStyleConfig({
  defaultProps: {
    size: "sm",
  },
  variants: {
    primary(props) {
      return {
        color: mode("white", "gray.50")(props),
        borderColor: "primary.600",
        _dark: {
          borderColor: "primary.500",
          backgroundColor: "primary.600",
        },
        borderWidth: 1,
        backgroundColor: "primary.500",
        _hover: {
          backgroundColor: "primary.600",
          _disabled: {
            background: "primary.500",
          },
          _dark: {
            backgroundColor: "primary.700",
          },
        },
      }
    },
    outline(props) {
      return {
        borderColor: props.colorScheme == "primary" ? "primary.600" : "gray.200",
        _dark: {
          borderColor: props.colorScheme == "primary" ? "primary.200" : "gray.700",
        },
        _hover: {
          _dark: {
            bg: props.colorScheme == "primary" ? "" : "gray.700",
          },
        },
        _active: {
          _dark: {
            bg: props.colorScheme == "primary" ? "" : "gray.800",
          },
        },
      }
    },
    solid(props) {
      let bgDark = "gray.800"
      if (props.colorScheme === "primary") {
        bgDark = ""
      } else {
        bgDark = defaultTheme.components.Button.variants?.solid(props).bg ?? ""
      }

      let bgHoverDark = "gray.700"
      if (props.colorScheme === "primary") {
        bgHoverDark = ""
      } else {
        bgHoverDark = defaultTheme.components.Button.variants?.solid(props)._hover.bg ?? ""
      }
      let bgActiveDark = "gray.700"
      if (props.colorScheme === "primary") {
        bgActiveDark = ""
      } else {
        bgActiveDark = defaultTheme.components.Button.variants?.solid(props)._active.bg ?? ""
      }

      return {
        _dark: {
          bg: bgDark,
        },
        _hover: {
          _dark: {
            bg: bgHoverDark,
          },
        },
        _active: {
          _dark: {
            bg: bgActiveDark,
          },
        },
      }
    },
    ["solid-outline"](props) {
      return {
        color: mode("gray.800", "gray.100")(props),
        borderColor: mode("gray.200", "gray.700")(props),
        borderWidth: 1,
        ".chakra-button__group[data-attached][data-orientation=horizontal] > &:not(:last-of-type)":
          { marginEnd: "-1px" },
        ".chakra-button__group[data-attached][data-orientation=vertical] > &:not(:last-of-type)": {
          marginBottom: "-1px",
        },
        backgroundColor: mode("gray.100", "gray.800")(props),
        _hover: {
          backgroundColor: mode("gray.200", "gray.700")(props),
        },
        _active: {
          backgroundColor: mode("gray.300", "gray.900")(props),
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
    proWorkspaceIDE(_props) {
      return {
        color: "primary.900",
        fontWeight: "semibold",
        bg: "primary.100",
        _hover: {
          bg: "primary.200",
        },
        _active: {
          bg: "primary.300",
        },
      }
    },
    ghost(props) {
      return {
        _hover: {
          _dark: {
            bg: props.colorScheme == "primary" ? "" : "gray.700",
          },
        },
        _active: {
          _dark: {
            bg: props.colorScheme == "primary" ? "" : "gray.800",
          },
        },
      }
    },
  },
})
