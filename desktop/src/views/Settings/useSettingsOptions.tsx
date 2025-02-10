import { useCallback, useMemo } from "react"
import { client } from "../../client"
import { Command } from "../../client/command"
import { TSettings, useChangeSettings } from "../../contexts"
import { ClearableInput } from "./ClearableInput"

export function useCLIFlagsOption() {
  const { settings, set } = useChangeSettings()

  const handleChanged = useCallback(
    (newValue: string) => {
      const value = newValue.trim()
      set("additionalCliFlags", value)
      client.setSetting("additionalCliFlags", value)
    },
    [set]
  )

  const input = useMemo(
    () => (
      <ClearableInput
        placeholder="Additional CLI Flags"
        defaultValue={settings.additionalCliFlags}
        onChange={handleChanged}
      />
    ),
    [handleChanged, settings.additionalCliFlags]
  )

  const helpText = useMemo(() => <>Set additional CLI Flags to use.</>, [])

  return { input, helpText }
}

export function useExtraEnvVarsOption() {
  const { settings, set } = useChangeSettings()

  const handleChanged = useCallback(
    (newValue: string) => {
      const value = newValue.trim()
      set("additionalEnvVars", value)
      Command.ADDITIONAL_ENV_VARS = value
    },
    [set]
  )

  const input = useMemo(
    () => (
      <ClearableInput
        placeholder="Additional Environment Variables"
        defaultValue={settings.additionalEnvVars}
        onChange={handleChanged}
      />
    ),
    [handleChanged, settings.additionalEnvVars]
  )

  const helpText = useMemo(
    () => (
      <>
        Set additional environment variables DevPod passes to all commands. Accepts a comma
        separated list, e.g. FOO=bar,BAZ=false
      </>
    ),
    []
  )

  return { input, helpText }
}

export function useDotfilesOption() {
  const { settings, set } = useChangeSettings()
  const updateOption = useCallback(
    (value: string) => {
      set("dotfilesUrl", value)
      client.setSetting("dotfilesUrl", value)
    },
    [set]
  )

  const input = useMemo(
    () => (
      <ClearableInput
        placeholder="Dotfiles repo URL"
        defaultValue={settings.dotfilesUrl}
        onChange={updateOption}
      />
    ),
    [settings.dotfilesUrl, updateOption]
  )

  return { input }
}

export function useSSHKeySignatureOption() {
  const { settings, set } = useChangeSettings()
  const updateOption = useCallback(
    (value: string) => {
      set("sshKeyPath", value)
      client.setSetting("sshKeyPath", value)
    },
    [set]
  )

  const input = useMemo(
    () => (
      <ClearableInput
        placeholder="SSH key path"
        defaultValue={settings.sshKeyPath}
        onChange={updateOption}
      />
    ),
    [settings.sshKeyPath, updateOption]
  )

  return { input }
}

export function useProxyOptions() {
  const { settings, set } = useChangeSettings()

  const handleChanged = useCallback(
    (s: keyof Pick<TSettings, "httpProxyUrl" | "httpsProxyUrl" | "noProxy">) =>
      (newValue: string) => {
        set(s, newValue)
        client.setSetting(s, newValue)
      },
    [set]
  )

  const input = useMemo(() => {
    return (
      <>
        <ClearableInput
          defaultValue={settings.httpProxyUrl}
          onChange={handleChanged("httpProxyUrl")}
          placeholder="Set HTTP_PROXY"
          label="HTTP"
        />
        <ClearableInput
          defaultValue={settings.httpsProxyUrl}
          onChange={handleChanged("httpsProxyUrl")}
          placeholder="Set HTTPS_PROXY"
          label="HTTPS"
        />
        <ClearableInput
          defaultValue={settings.noProxy}
          onChange={handleChanged("noProxy")}
          placeholder="Set NO_PROXY"
          label="NO_PROXY"
        />
      </>
    )
  }, [handleChanged, settings.httpProxyUrl, settings.httpsProxyUrl, settings.noProxy])

  const helpText = useMemo(
    () => (
      <>
        Set HTTP(S) proxy configuration. These settings will only be used by DevPod itself and not
        be available within your workspace.
      </>
    ),
    []
  )

  return { input, helpText }
}
