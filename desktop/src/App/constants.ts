import { BoxProps } from "@chakra-ui/react"
import { isLinux, isMacOS, isWindows } from "../lib"

export const showTitleBar = isMacOS || isLinux || isWindows
export const titleBarSafeArea: BoxProps["height"] = showTitleBar ? isWindows ? "6" : "12" : 0
