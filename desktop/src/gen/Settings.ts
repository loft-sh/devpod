// This file was generated by [ts-rs](https://github.com/Aleph-Alpha/ts-rs). Do not edit this file manually.
import type { ColorMode } from "./ColorMode"
import type { SidebarPosition } from "./SidebarPosition"
import type { Zoom } from "./Zoom"

export interface Settings {
  sidebarPosition: SidebarPosition
  debugFlag: boolean
  partyParrot: boolean
  fixedIDE: boolean
  zoom: Zoom
  transparency: boolean
  autoUpdate: boolean
  experimental_multiDevcontainer: boolean
  experimental_fleet: boolean
  experimental_jupyterNotebooks: boolean
  experimental_vscodeInsiders: boolean
  experimental_devPodPro: boolean
  experimental_colorMode: ColorMode
  additionalCliFlags: string
  additionalEnvVars: string
  dotfilesUrl: string
  sshKeyPath: string
  httpProxyUrl: string
  httpsProxyUrl: string
  noProxy: string
}
