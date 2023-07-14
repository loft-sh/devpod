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
    source: "https://github.com/microsoft/vscode-remote-try-go",
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
  {
    image: DotnetcoreSvg,
    source: "https://github.com/microsoft/vscode-remote-try-dotnet",
    name: "Try Dotnet",
  },
]
