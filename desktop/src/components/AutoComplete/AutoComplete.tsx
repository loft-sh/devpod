import {
  Box,
  Fade,
  Icon,
  Input,
  InputGroup,
  InputRightElement,
  useColorModeValue,
  useToken,
} from "@chakra-ui/react"
import { Combobox } from "@headlessui/react"
import { forwardRef, useEffect, useRef, useState } from "react"
import { AiOutlineCaretRight } from "react-icons/ai"

type TAutoCompleteOption = Readonly<{
  key: string
  label: string
}>
type TAutoCompleteProps = Readonly<{
  options: readonly TAutoCompleteOption[]
  onChange?: (value: string) => void
  onBlur?: () => void
  value?: string
  defaultValue?: string
  placeholder?: string
  name?: string
}>

/* 
 * Can be integrated with `react-hook-form` like this:
 * ```tsx
    const {  handleSubmit, control } = useForm()

    <form onSubmit={handleSubmit(onSubmit)}>
      <Controller
        name="auto"
        control={control}
        render={({ field }) => <AutoComplete options={options} {...field} />}
      />
      <button type="submit">Submit</button>
    </form>
  ```
 */
export const AutoComplete = forwardRef<HTMLElement, TAutoCompleteProps>(function InnerAutoComplete(
  { name, placeholder, options, defaultValue, value, onChange, onBlur },
  ref
) {
  const openButtonRef = useRef<HTMLButtonElement>(null)
  const optionsBackgroundColor = useColorModeValue("gray.100", "gray.800")
  const optionsZIndex = useToken("zIndices", "dropdown")
  const [query, setQuery] = useState("")

  useEffect(() => {
    // set value initially
    if (value) {
      onChange?.(value)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const filteredOptions =
    query === ""
      ? options
      : options.filter((option) => {
          return option.key.toLowerCase().includes(query.toLowerCase())
        })

  function handleInputFocused(isOpen: boolean) {
    return () => {
      if (!isOpen) {
        openButtonRef.current?.click()
      }
    }
  }

  return (
    <Combobox<string>
      ref={ref}
      name={name}
      value={value}
      defaultValue={defaultValue}
      onChange={onChange}>
      {({ open: isOpen }) => (
        <Box position="relative" zIndex={""}>
          <InputGroup>
            <Input
              onBlur={onBlur}
              spellCheck={false}
              as={Combobox.Input}
              placeholder={placeholder}
              onChange={(event) => {
                setQuery(event.target.value)
                onChange?.(event.target.value)
              }}
              onClick={handleInputFocused(isOpen)}
            />
            <InputRightElement>
              <Box
                onClick={(e) => {
                  e.preventDefault()
                  e.stopPropagation()
                }}>
                <Combobox.Button ref={openButtonRef}>
                  <Icon
                    boxSize={4}
                    transition={"transform .2s"}
                    transform={isOpen ? "rotate(90deg)" : ""}
                    as={AiOutlineCaretRight}
                  />
                </Combobox.Button>
              </Box>
            </InputRightElement>
          </InputGroup>
          <Combobox.Options
            style={{
              position: "absolute",
              width: "100%",
              zIndex: optionsZIndex,
            }}>
            {({ open: isOpen }) => (
              <Box
                maxHeight="48"
                overflowY="auto"
                backgroundColor={optionsBackgroundColor}
                padding="2"
                borderRadius="md">
                <Fade in={isOpen}>
                  {query.length > 0 && !filteredOptions.find((o) => o.label === query) && (
                    <Option option={{ key: query, label: query }} />
                  )}
                  {filteredOptions.map((option) => (
                    <Option key={option.key} option={option} />
                  ))}
                </Fade>
              </Box>
            )}
          </Combobox.Options>
        </Box>
      )}
    </Combobox>
  )
})

function Option({ option }: { option: TAutoCompleteOption }) {
  const activeOptionBackgroundColor = useColorModeValue("gray.200", "gray.700")

  return (
    <Combobox.Option style={{ listStyleType: "none" }} key={option.key} value={option.key}>
      {({ active }) => (
        <Box
          padding="2"
          borderRadius="md"
          backgroundColor={active ? activeOptionBackgroundColor : undefined}>
          {option.label}
        </Box>
      )}
    </Combobox.Option>
  )
}
