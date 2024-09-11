import { CloseIcon } from "@chakra-ui/icons"
import { IconButton, Input, InputGroup, InputLeftAddon, InputRightElement } from "@chakra-ui/react"
import { FocusEvent, KeyboardEvent, ReactNode, useCallback, useRef, useState } from "react"

type TClearableInputProps = Readonly<{
  defaultValue: string
  placeholder: string
  label?: ReactNode
  onChange: (newValue: string) => void
}>
export function ClearableInput({
  defaultValue,
  placeholder,
  label,
  onChange,
}: TClearableInputProps) {
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
