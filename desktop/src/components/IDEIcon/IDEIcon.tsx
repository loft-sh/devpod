import {
  Box,
  BoxProps,
  Icon,
  IconProps,
  Image,
  useColorMode,
  useColorModeValue,
  useToken,
} from "@chakra-ui/react"
import { HiBeaker } from "react-icons/hi2"
import {
  CLionSvg,
  CodiumSvg,
  CursorSvg,
  DataSpellSvg,
  FleetSvg,
  GolandSvg,
  IntelliJSvg,
  JupyterNotebookDarkSvg,
  JupyterNotebookSvg,
  NoneSvg,
  NoneSvgDark,
  PHPStormSvg,
  PositronSvg,
  PycharmSvg,
  RStudioSvg,
  RiderSvg,
  RubyMineSvg,
  RustRoverSvg,
  VSCodeBrowser,
  VSCodeInsidersSvg,
  VSCodeSvg,
  WebstormSvg,
  ZedDarkSvg,
  ZedSvg,
} from "../../images"
import { TIDE } from "../../types"
import { useMemo } from "react"

const SIZES: Record<NonNullable<TIDEIconProps["size"]>, IconProps> = {
  sm: {
    boxSize: 3,
    padding: "1px",
  },
  md: {
    boxSize: 6,
    padding: "3px",
  },
}

const IDE_ICONS: Record<string, string> = {
  none: NoneSvg,
  vscode: VSCodeSvg,
  "vscode-insiders": VSCodeInsidersSvg,
  openvscode: VSCodeBrowser,
  intellij: IntelliJSvg,
  goland: GolandSvg,
  rustrover: RustRoverSvg,
  pycharm: PycharmSvg,
  phpstorm: PHPStormSvg,
  clion: CLionSvg,
  rubymine: RubyMineSvg,
  rider: RiderSvg,
  webstorm: WebstormSvg,
  dataspell: DataSpellSvg,
  fleet: FleetSvg,
  jupyternotebook: JupyterNotebookSvg,
  jupyternotebook_dark: JupyterNotebookDarkSvg,
  cursor: CursorSvg,
  positron: PositronSvg,
  codium: CodiumSvg,
  zed: ZedSvg,
  zed_dark: ZedDarkSvg,
  rstudio: RStudioSvg,
}

type TIDEIconProps = Readonly<{ ide: TIDE; size?: "sm" | "md" }> & BoxProps
export function IDEIcon({ ide, size = "md", ...boxProps }: TIDEIconProps) {
  const experimentalIconSizeProps = SIZES[size]
  const primaryColorDarkToken = useColorModeValue("primary.800", "primary.400")
  const primaryColorDark = useToken("colors", primaryColorDarkToken)
  const primaryColorLightToken = useColorModeValue("primary.400", "primary.800")
  const primaryColorLight = useToken("colors", primaryColorLightToken)
  const backgroundColor = useColorModeValue("white", "gray.700")
  const { colorMode } = useColorMode()

  const experimentalIconStylingProps =
    size === "sm"
      ? {
          color: primaryColorDark,
        }
      : {
          boxShadow: `inset 0px 0px 0px 1px ${primaryColorDark}55`,
          background:
            colorMode === "light"
              ? `linear-gradient(135deg, ${primaryColorLight}55 50%, ${primaryColorDark}55, ${primaryColorDark}88)`
              : `linear-gradient(135deg, ${primaryColorDark}55 50%, ${primaryColorLight}55, ${primaryColorLight}88)`,
          color: `${primaryColorDark}CC`,
        }
  const fallbackIcon = colorMode === "light" ? NoneSvg : NoneSvgDark

  const icon = useMemo(() => {
    if (colorMode === "light") {
      return IDE_ICONS[ide.name!] ?? ide.icon
    } else {
      const darkIcon = IDE_ICONS[ide.name! + "_dark"] ?? ide.iconDark
      if (darkIcon) {
        return darkIcon
      }

      // fall back to regular icon
      return IDE_ICONS[ide.name!] ?? ide.icon
    }
  }, [colorMode, ide])

  return (
    <Box width="full" height="full" position="relative">
      <Image src={icon ?? fallbackIcon} {...boxProps} />
      {ide.experimental && (
        <>
          <Box
            position="absolute"
            bottom="0"
            right="0"
            zIndex="docked"
            borderRadius="full"
            boxSize={experimentalIconSizeProps.boxSize}
            backgroundColor={backgroundColor}
          />
          <Icon
            position="absolute"
            bottom="0"
            right="0"
            zIndex="docked"
            borderRadius="full"
            as={HiBeaker}
            {...experimentalIconSizeProps}
            {...experimentalIconStylingProps}
          />
        </>
      )}
    </Box>
  )
}
