import { BoxProps } from "@chakra-ui/react"
import {
  AWSSvg,
  AWSWhiteSvg,
  AzureSvg,
  CivoSvg,
  DigitalOceanSvg,
  DockerSvg,
  GCloudSvg,
  KubernetesSvg,
  SSHSvg,
} from "./images"

export const STATUS_BAR_HEIGHT: NonNullable<BoxProps["height"]> = "8"
export const SIDEBAR_WIDTH: BoxProps["width"] = "15rem"
export const RECOMMENDED_PROVIDER_SOURCES = [
  // generic
  { image: DockerSvg, imageDarkMode: undefined, name: "docker", group: "generic" },
  { image: KubernetesSvg, imageDarkMode: undefined, name: "kubernetes", group: "generic" },
  { image: SSHSvg, imageDarkMode: undefined, name: "ssh", group: "generic" },
  // cloud
  { image: AWSSvg, imageDarkMode: AWSWhiteSvg, name: "aws", group: "cloud" },
  { image: GCloudSvg, imageDarkMode: undefined, name: "gcloud", group: "cloud" },
  { image: AzureSvg, imageDarkMode: undefined, name: "azure", group: "cloud" },
  { image: DigitalOceanSvg, imageDarkMode: undefined, name: "digitalocean", group: "cloud" },
  { image: CivoSvg, imageDarkMode: undefined, name: "civo", group: "cloud" },
] as const

export const WORKSPACE_SOURCE_BRANCH_DELIMITER = "@"
export const WORKSPACE_SOURCE_COMMIT_DELIMITER = "@sha256:"
export const WORKSPACE_STATUSES = ["Running", "Stopped", "Busy", "NotFound"] as const
