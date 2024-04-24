import { CloseIcon } from "@chakra-ui/icons"
import {
  Code,
  IconButton,
  Input,
  InputGroup,
  InputRightElement,
  Link,
  Switch,
} from "@chakra-ui/react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { FocusEvent, KeyboardEvent, useCallback, useMemo, useRef, useState } from "react"
import { client } from "../../client"
import { useChangeSettings } from "../../contexts"
import { QueryKeys } from "../../queryKeys"
import { TContextOptionName } from "../../types"
import { Command } from "../../client/command"

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
  const updateOption = useCallback(
    (value: string) => {
      set("additionalCliFlags", value)
      client.setSetting("additionalCliFlags", value)
    },
    [set]
  )
  const [hasFocus, setHasFocus] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  const handleBlur = useCallback(
    (e: FocusEvent<HTMLInputElement>) => {
      const value = e.target.value.trim()
      updateOption(value)
      setHasFocus(false)
    },
    [updateOption]
  )

  const handleKeyUp = useCallback((e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key !== "Enter") return

    e.currentTarget.blur()
  }, [])

  const handleFocus = useCallback(() => {
    setHasFocus(true)
  }, [])

  const handleClearDevPodCLIFlags = useCallback(() => {
    const el = inputRef.current
    if (!el) return

    el.value = ""
  }, [])

  const input = useMemo(
    () => (
      <InputGroup maxWidth="72">
        <Input
          ref={inputRef}
          spellCheck={false}
          placeholder="Additional CLI Flags"
          defaultValue={settings.additionalCliFlags}
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
            onClick={handleClearDevPodCLIFlags}
          />
        </InputRightElement>
      </InputGroup>
    ),
    [
      settings.additionalCliFlags,
      handleBlur,
      handleKeyUp,
      handleFocus,
      hasFocus,
      handleClearDevPodCLIFlags,
    ]
  )

  const helpText = useMemo(() => <>Set additional CLI Flags to use.</>, [])

  return { input, helpText }
}

export function useAgentURLOption() {
  const { options, updateOption } = useContextOptions()
  const [hasFocus, setHasFocus] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  const handleBlur = useCallback(
    (e: FocusEvent<HTMLInputElement>) => {
      const value = e.target.value.trim()
      updateOption({ option: "AGENT_URL", value })
      setHasFocus(false)
    },
    [updateOption]
  )

  const handleKeyUp = useCallback((e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key !== "Enter") return

    e.currentTarget.blur()
  }, [])

  const handleFocus = useCallback(() => {
    setHasFocus(true)
  }, [])

  const handleClearDevPodAgent = useCallback(() => {
    const el = inputRef.current
    if (!el) return

    el.value = ""
  }, [])

  const input = useMemo(
    () => (
      <InputGroup maxWidth="72">
        <Input
          ref={inputRef}
          spellCheck={false}
          placeholder="Override Agent URL"
          defaultValue={options?.AGENT_URL.value ?? undefined}
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
            onClick={handleClearDevPodAgent}
          />
        </InputRightElement>
      </InputGroup>
    ),
    [
      options?.AGENT_URL.value,
      handleBlur,
      handleKeyUp,
      handleFocus,
      hasFocus,
      handleClearDevPodAgent,
    ]
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
        <Link onClick={() => client.openLink("https://devpod.sh/docs/other-topics/telemetry")}>
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
  const [hasFocus, setHasFocus] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  const handleBlur = useCallback(
    (e: FocusEvent<HTMLInputElement>) => {
      const value = e.target.value.trim()
      set("additionalEnvVars", value)
      Command.ADDITIONAL_ENV_VARS = value
      setHasFocus(false)
    },
    [set]
  )

  const handleKeyUp = useCallback((e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key !== "Enter") return

    e.currentTarget.blur()
  }, [])

  const handleFocus = useCallback(() => {
    setHasFocus(true)
  }, [])

  const handleClear = useCallback(() => {
    const el = inputRef.current
    if (!el) return

    el.value = ""
  }, [])

  const input = useMemo(
    () => (
      <InputGroup maxWidth="72">
        <Input
          ref={inputRef}
          spellCheck={false}
          placeholder="Additional Environment Variables"
          defaultValue={settings.additionalEnvVars}
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
            onClick={handleClear}
          />
        </InputRightElement>
      </InputGroup>
    ),
    [settings.additionalEnvVars, handleBlur, handleKeyUp, handleFocus, hasFocus, handleClear]
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
      set("dotfilesURL", value)
      client.setSetting("dotfilesURL", value)
    },
    [set]
  )
  const [hasFocus, setHasFocus] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  const handleBlur = useCallback(
    (e: FocusEvent<HTMLInputElement>) => {
      const value = e.target.value.trim()
      updateOption(value)
      setHasFocus(false)
    },
    [updateOption]
  )

  const handleKeyUp = useCallback((e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key !== "Enter") return

    e.currentTarget.blur()
  }, [])

  const handleFocus = useCallback(() => {
    setHasFocus(true)
  }, [])

  const handleClearDevPodDotfiles = useCallback(() => {
    const el = inputRef.current
    if (!el) return

    el.value = ""
  }, [])

  const input = useMemo(
    () => (
      <InputGroup maxWidth="72">
        <Input
          ref={inputRef}
          spellCheck={false}
          placeholder="Dotfiles repo URL"
          defaultValue={settings.dotfilesURL}
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
            onClick={handleClearDevPodDotfiles}
          />
        </InputRightElement>
      </InputGroup>
    ),
    [
      settings.dotfilesURL,
      handleBlur,
      handleKeyUp,
      handleFocus,
      hasFocus,
      handleClearDevPodDotfiles,
    ]
  )

  return { input }
}
