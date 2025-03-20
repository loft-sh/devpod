import { createContext, useContext } from "react"
import { Settings } from "@/gen"

export type TSettings = Settings
export type TSetting = keyof TSettings

export type TSettingsContext = Readonly<{
  settings: TSettings
  set<TKey extends keyof TSettings>(setting: TKey, value: TSettings[TKey]): void
}>

export const SettingsContext = createContext<TSettingsContext>(null!)
export function useSettings() {
  return useContext(SettingsContext).settings
}

export function useChangeSettings() {
  return useContext(SettingsContext)
}
