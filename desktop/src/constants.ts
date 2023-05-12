import { BoxProps } from "@chakra-ui/react"
import {
  AWSSvg,
  AzureSvg,
  DigitalOceanSvg,
  DockerPng,
  GCloudSvg,
  KubernetesSvg,
  SSHPng,
} from "./images"

export const STATUS_BAR_HEIGHT: NonNullable<BoxProps["height"]> = "8"
export const SIDEBAR_WIDTH: BoxProps["width"] = "15rem"
export const RECOMMENDED_PROVIDER_SOURCES = [
  // generic
  { image: DockerPng, name: "docker", group: "generic" },
  { image: KubernetesSvg, name: "kubernetes", group: "generic" },
  { image: SSHPng, name: "ssh", group: "generic" },
  // cloud
  { image: AWSSvg, name: "aws", group: "cloud" },
  { image: GCloudSvg, name: "gcloud", group: "cloud" },
  { image: AzureSvg, name: "azure", group: "cloud" },
  { image: DigitalOceanSvg, name: "digitalocean", group: "cloud" },
] as const
