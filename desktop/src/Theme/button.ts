import { defineStyleConfig } from "@chakra-ui/react"
import { mode } from "@chakra-ui/theme-tools"

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
      return {
        _dark: {
          bg: "gray.800",
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
    proWorkspaceIDE(props) {
      return {
        color: mode("white", "black")(props),
        fontWeight: "semibold",
        bg: mode("gray.600", "gray.300")(props),
        _hover: {
          _dark: {
            bg: "gray.400",
          },
          bg: "gray.700",
        },
        _active: {
          bg: "gray.800",
          _dark: {
            bg: "gray.500",
          },
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
