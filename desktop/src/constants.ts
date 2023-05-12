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
  { image: DockerPng, name: "docker" },
  { image: SSHPng, name: "ssh" },
  { image: KubernetesSvg, name: "kubernetes" },
  { image: AWSSvg, name: "aws" },
  { image: GCloudSvg, name: "gcloud" },
  { image: AzureSvg, name: "azure" },
  { image: DigitalOceanSvg, name: "digitalocean" },
] as const
