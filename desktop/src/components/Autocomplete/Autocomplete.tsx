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
import { forwardRef, ReactNode, useState } from "react"
import { AiOutlineCaretRight } from "react-icons/ai"

type TAutoCompleteOption = Readonly<{
  key: string
  label: ReactNode
}>
type TAutoCompleteProps = Readonly<{
  options: readonly TAutoCompleteOption[]
  onChange?: (value: TAutoCompleteOption) => void
  defaultValue?: TAutoCompleteOption
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
export const AutoComplete = forwardRef<HTMLElement, TAutoCompleteProps>(function InnerAutocomlete(
  { name, options, defaultValue, onChange },
  ref
) {
  const optionsBackgroundColor = useColorModeValue("gray.100", "gray.800")
  const optionsZIndex = useToken("zIndices", "dropdown")
  const [query, setQuery] = useState("")

  const filteredOptions =
    query === ""
      ? options
      : options.filter((option) => {
          return option.key.toLowerCase().includes(query.toLowerCase())
        })

  return (
    <Combobox<TAutoCompleteOption>
      ref={ref}
      name={name}
      defaultValue={defaultValue}
      onChange={onChange}>
      {({ open: isOpen }) => (
        <Box position="relative" zIndex={""}>
          <InputGroup>
            <Input
              spellCheck={false}
              as={Combobox.Input}
              onChange={(event) => setQuery(event.target.value)}
            />
            <InputRightElement>
              <Combobox.Button>
                <Icon
                  boxSize={4}
                  transition={"transform .2s"}
                  transform={isOpen ? "rotate(90deg)" : ""}
                  as={AiOutlineCaretRight}
                />
              </Combobox.Button>
            </InputRightElement>
          </InputGroup>
          <Combobox.Options style={{ position: "absolute", width: "100%", zIndex: optionsZIndex }}>
            {({ open: isOpen }) => (
              <Box backgroundColor={optionsBackgroundColor} padding="2" borderRadius="md">
                <Fade in={isOpen}>
                  {query.length > 0 && <Option option={{ key: query, label: query }} />}
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
