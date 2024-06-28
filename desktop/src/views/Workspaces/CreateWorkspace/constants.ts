import {
  CppSvg,
  DotnetcoreSvg,
  GoSvg,
  GoSvgDark,
  JavaSvg,
  NodejsSvg,
  PhpSvg,
  PhpSvgDark,
  PythonSvg,
  RubySvg,
  RustSvg,
  RustSvgDark,
} from "../../../images"

export const WORKSPACE_EXAMPLES = [
  {
    image: PythonSvg,
    source: "https://github.com/microsoft/vscode-remote-try-python",
    name: "Try Python",
  },
  {
    image: NodejsSvg,
    source: "https://github.com/microsoft/vscode-remote-try-node",
    name: "Try Node",
  },
  {
    image: GoSvg,
    imageDark: GoSvgDark,
    source: "https://github.com/loft-sh/devpod-example-go",
    name: "Try Go",
  },
  {
    image: RustSvg,
    imageDark: RustSvgDark,
    source: "https://github.com/microsoft/vscode-remote-try-rust",
    name: "Try Rust",
  },
  {
    image: JavaSvg,
    source: "https://github.com/microsoft/vscode-remote-try-java",
    name: "Try Java",
  },
  {
    image: PhpSvg,
    imageDark: PhpSvgDark,
    source: "https://github.com/microsoft/vscode-remote-try-php",
    name: "Try PHP",
  },
  {
    image: CppSvg,
    source: "https://github.com/microsoft/vscode-remote-try-cpp",
    name: "Try C++",
  },
]

export const COMMUNITY_WORKSPACE_EXAMPLES = [
  {
    image: RubySvg,
    imageDark: RubySvg,
    source: "https://github.com/loft-sh/devpod-quickstart-ruby",
    name: "Try Ruby",
  },
  {
    image: DotnetcoreSvg,
    source: "https://github.com/microsoft/vscode-remote-try-dotnet",
    name: "Try Dotnet",
  },
]
