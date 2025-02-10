import {
  createContext,
  ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react"
import { client } from "../client"
import { Settings } from "../gen"
import { getKeys, LocalStorageToFileMigrationBackend, Store } from "../lib"
import { TUnsubscribeFn } from "../types"

export type TSettings = Settings
type TSetting = keyof TSettings

type TSettingsContext = Readonly<{
  settings: TSettings
  set<TKey extends keyof TSettings>(setting: TKey, value: TSettings[TKey]): void
}>

const SettingsContext = createContext<TSettingsContext>(null!)

const initialSettings: TSettings = {
  sidebarPosition: "left",
  debugFlag: false,
  partyParrot: false,
  fixedIDE: false,
  zoom: "md",
  transparency: false,
  autoUpdate: true,
  additionalCliFlags: "",
  additionalEnvVars: "",
  dotfilesUrl: "",
  sshKeyPath: "",
  httpProxyUrl: "",
  httpsProxyUrl: "",
  noProxy: "",

  experimental_colorMode: "light",
  experimental_multiDevcontainer: false,
  experimental_fleet: true,
  experimental_jupyterNotebooks: true,
  experimental_vscodeInsiders: true,
  experimental_cursor: true,
  experimental_positron: true,
  experimental_jupyterDesktop: true,
  experimental_zed: true,
  experimental_codium: true,
  experimental_marimo: true,
  experimental_devPodPro: false,
}
function getSettingKeys(): readonly TSetting[] {
  return getKeys(initialSettings)
}

// WARN: needs to match the filename on the rust side
const SETTING_STORE_KEY = "settings"
const settingsStore = new Store<Record<TSetting, string | boolean | unknown>>(
  new LocalStorageToFileMigrationBackend(SETTING_STORE_KEY)
)

export function SettingsProvider({ children }: Readonly<{ children?: ReactNode }>) {
  const [settings, setSettings] = useState(initialSettings)

  useEffect(() => {
    ;(async () => {
      const initialOptions = await Promise.all(
        getSettingKeys().map((option) =>
          settingsStore
            .get(option)
            .then((value) => [option, value ?? initialSettings[option]] as const)
            .catch(() => [option, false] as const)
        )
      )
      setSettings(
        initialOptions.reduce((acc, [key, value]) => ({ ...acc, [key]: value }), initialSettings)
      )
    })()
  }, [])

  useEffect(() => {
    const subscriptions: TUnsubscribeFn[] = []

    for (const setting of getSettingKeys()) {
      subscriptions.push(
        settingsStore.subscribe(setting, (newValue) =>
          setSettings((current) => ({ ...current, [setting]: newValue }))
        )
      )
    }

    return () => {
      for (const unsubscribe of subscriptions) {
        unsubscribe()
      }
    }
  }, [])

  useEffect(() => {
    client.setSetting("debugFlag", settings.debugFlag)
  }, [settings.debugFlag])

  useEffect(() => {
    client.setSetting("additionalCliFlags", settings.additionalCliFlags)
  }, [settings.additionalCliFlags])

  useEffect(() => {
    client.setSetting("dotfilesUrl", settings.dotfilesUrl)
  }, [settings.dotfilesUrl])

  useEffect(() => {
    client.setSetting("additionalEnvVars", settings.additionalEnvVars)
  }, [settings.additionalEnvVars])

  useEffect(() => {
    client.setSetting("httpProxyUrl", settings.httpProxyUrl)
    client.setSetting("httpsProxyUrl", settings.httpsProxyUrl)
    client.setSetting("noProxy", settings.noProxy)
  }, [settings.httpProxyUrl, settings.httpsProxyUrl, settings.noProxy])

  const set = useCallback<TSettingsContext["set"]>((key, value) => {
    settingsStore.set(key, value)
  }, [])

  const value = useMemo(() => ({ settings, set }), [set, settings])

  return <SettingsContext.Provider value={value}>{children}</SettingsContext.Provider>
}

export function useSettings() {
  return useContext(SettingsContext).settings
}

export function useChangeSettings() {
  return useContext(SettingsContext)
}
