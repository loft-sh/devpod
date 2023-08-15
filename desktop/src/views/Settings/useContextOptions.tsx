import { CloseIcon } from "@chakra-ui/icons"
import { Code, IconButton, Input, InputGroup, InputRightElement } from "@chakra-ui/react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { FocusEvent, KeyboardEvent, useCallback, useMemo, useRef, useState } from "react"
import { client } from "../../client"
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
