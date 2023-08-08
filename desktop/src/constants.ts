import { BoxProps } from "@chakra-ui/react"
import {
  AWSSvg,
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
  { image: DockerSvg, name: "docker", group: "generic" },
  { image: KubernetesSvg, name: "kubernetes", group: "generic" },
  { image: SSHSvg, name: "ssh", group: "generic" },
  // cloud
  { image: AWSSvg, name: "aws", group: "cloud" },
  { image: GCloudSvg, name: "gcloud", group: "cloud" },
  { image: AzureSvg, name: "azure", group: "cloud" },
  { image: DigitalOceanSvg, name: "digitalocean", group: "cloud" },
  { image: CivoSvg, name: "civo", group: "cloud" },
  // WARN: These are currently unstable
  // { image: TerraformSvg, name: "terraform", group: "cloud" },
] as const

export const WORKSPACE_SOURCE_BRANCH_DELIMITER = "@"
export const WORKSPACE_SOURCE_COMMIT_DELIMITER = "@sha256:"
