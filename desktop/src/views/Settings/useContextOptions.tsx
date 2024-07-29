import { CloseIcon } from "@chakra-ui/icons"
import {
  Code,
  IconButton,
  Input,
  InputGroup,
  InputLeftAddon,
  InputRightElement,
  Link,
  Switch,
} from "@chakra-ui/react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { FocusEvent, KeyboardEvent, ReactNode, useCallback, useMemo, useRef, useState } from "react"
import { client } from "../../client"
import { Command } from "../../client/command"
import { TSettings, useChangeSettings } from "../../contexts"
import { QueryKeys } from "../../queryKeys"
import { TContextOptionName } from "../../types"

const DEFAULT_DEVPOD_AGENT_URL = "https://github.com/loft-sh/devpod/releases/latest/download/"

export function useContextOptions() {
  const queryClient = useQueryClient()
  const { data: options } = useQuery({
    queryKey: QueryKeys.CONTEXT_OPTIONS,
    queryFn: async () => (await client.context.listOptions()).unwrap(),
  })
  const { mutate: updateOption } = useMutation({
    mutationFn: async ({ option, value }: { option: TContextOptionName; value: string }) => {
      ;(await client.context.setOption(option, value)).unwrap()
    },
    onSettled: () => {
      queryClient.invalidateQueries(QueryKeys.CONTEXT_OPTIONS)
    },
  })

  return useMemo(
    () => ({
      options,
      updateOption,
    }),
    [options, updateOption]
  )
}
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

export function useAgentURLOption() {
  const { options, updateOption } = useContextOptions()

  const handleChanged = useCallback(
    (newValue: string) => {
      const value = newValue.trim()
      updateOption({ option: "AGENT_URL", value })
    },
    [updateOption]
  )

  const input = useMemo(
    () => (
      <ClearableInput
        placeholder="Override Agent URL"
        defaultValue={options?.AGENT_URL.value ?? ""}
        onChange={handleChanged}
      />
    ),
    [handleChanged, options?.AGENT_URL.value]
  )

  const helpText = useMemo(
    () => (
      <>
        Set the Agent URL. If you leave this empty, it will be pulled from{" "}
        <Code>{DEFAULT_DEVPOD_AGENT_URL}</Code>
      </>
    ),
    []
  )

  return { input, helpText }
}
export function useTelemetryOption() {
  const { options, updateOption } = useContextOptions()

  const input = useMemo(
    () => (
      <Switch
        isChecked={options?.TELEMETRY.value === "true"}
        onChange={(e) => updateOption({ option: "TELEMETRY", value: e.target.checked.toString() })}
      />
    ),
    [options?.TELEMETRY.value, updateOption]
  )

  const helpText = useMemo(
    () => (
      <>
        Telemetry plays an important role in improving DevPod for everyone.{" "}
        <strong>We never collect any actual values, only anonymized metadata!</strong> For an
        in-depth explanation, please refer to the{" "}
        <Link onClick={() => client.open("https://devpod.sh/docs/other-topics/telemetry")}>
          documentation
        </Link>
      </>
    ),
    []
  )

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

type TClearableInputProps = Readonly<{
  defaultValue: string
  placeholder: string
  label?: ReactNode
  onChange: (newValue: string) => void
}>
function ClearableInput({ defaultValue, placeholder, label, onChange }: TClearableInputProps) {
  const [hasFocus, setHasFocus] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  const handleBlur = useCallback(
    (e: FocusEvent<HTMLInputElement>) => {
      const value = e.target.value.trim()
      onChange(value)
      setHasFocus(false)
    },
    [onChange]
  )

  const handleKeyUp = useCallback((e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key !== "Enter") return

    e.currentTarget.blur()
  }, [])

  const handleFocus = useCallback(() => {
    setHasFocus(true)
  }, [])

  const handleClearClicked = useCallback(() => {
    const el = inputRef.current
    if (!el) return

    el.value = ""
  }, [])

  return (
    <InputGroup maxWidth="96">
      {label && <InputLeftAddon>{label}</InputLeftAddon>}
      <Input
        ref={inputRef}
        spellCheck={false}
        placeholder={placeholder}
        defaultValue={defaultValue}
        onBlur={handleBlur}
        onKeyUp={handleKeyUp}
        onFocus={handleFocus}
      />
      <InputRightElement>
        <IconButton
          visibility={hasFocus ? "visible" : "hidden"}
          size="xs"
          borderRadius="full"
          icon={<CloseIcon />}
          aria-label="clear"
          onMouseDown={(e) => {
            // needed to prevent losing focus from input
            e.stopPropagation()
            e.preventDefault()
          }}
          onClick={handleClearClicked}
        />
      </InputRightElement>
    </InputGroup>
  )
}
