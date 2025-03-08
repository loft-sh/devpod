import {
  Box,
  HStack,
  IconButton,
  Input,
  InputGroup,
  InputLeftElement,
  InputRightElement,
  Tooltip,
  useColorModeValue,
} from "@chakra-ui/react"
import { ArrowDown, ArrowUp, MatchCase, Search, WholeWord } from "@/icons"
import { ReactElement, useEffect, useRef, useState } from "react"
import { TSearchOptions } from "@/components/Terminal/useTerminalSearch"

type TTerminalSearchBarProps = {
  prevSearchResult: VoidFunction
  nextSearchResult: VoidFunction
  totalSearchResults: number
  activeSearchResult: number
  onUpdateSearchOptions: (searchOptions: TSearchOptions) => void
  paddingX?: number
  paddingY?: number
}

export function TerminalSearchBar({
  prevSearchResult,
  nextSearchResult,
  totalSearchResults,
  activeSearchResult,
  onUpdateSearchOptions,
  paddingY,
  paddingX,
}: TTerminalSearchBarProps) {
  const bgColor = useColorModeValue("white", "gray.800")
  const [searchString, setSearchString] = useState<string | undefined>(undefined)
  const [debouncedSearchString, setDebouncedSearchString] = useState<string | undefined>(undefined)
  const [caseSensitive, setCaseSensitive] = useState<boolean>(false)
  const [wholeWordSearch, setWholeWordSearch] = useState<boolean>(false)

  const searchInputRef = useRef<HTMLInputElement | null>(null)

  // Debounce to prevent stutter when having a huge amount of results.
  useEffect(() => {
    // Sneaky heuristic:
    // If we have more than two characters, we're likely to have a more sane amount of results, so we can skip debouncing.
    const len = searchString?.length ?? 0
    if (len > 2) {
      setDebouncedSearchString(searchString)

      return
    }

    const timeout = setTimeout(() => {
      setDebouncedSearchString(searchString)
    }, 200)

    return () => clearTimeout(timeout)
  }, [searchString])

  // Pass on the search options to the outside world. We do it like this so the debouncing works nicely.
  useEffect(() => {
    onUpdateSearchOptions({ searchString: debouncedSearchString, caseSensitive, wholeWordSearch })
  }, [debouncedSearchString, wholeWordSearch, caseSensitive, onUpdateSearchOptions])

  return (
    <HStack w={"full"} alignItems={"center"} paddingX={paddingX} paddingY={paddingY}>
      <InputGroup>
        <InputLeftElement cursor={"text"} onClick={() => searchInputRef.current?.focus()}>
          <Search boxSize={5} color={"text.tertiary"} />
        </InputLeftElement>
        <Input
          ref={searchInputRef}
          value={searchString ?? ""}
          placeholder={"Search..."}
          spellCheck={false}
          bg={bgColor}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              if (e.shiftKey) {
                prevSearchResult()
              } else {
                nextSearchResult()
              }
            }
          }}
          onChange={(e) => {
            setSearchString(e.target.value ? e.target.value : undefined)
          }}
        />
        <InputRightElement w={"fit-content"} paddingX={"4"}>
          <HStack alignItems={"center"} w={"fit-content"}>
            <ToggleButton
              label={"Case sensitive"}
              icon={<MatchCase boxSize={5} />}
              value={caseSensitive}
              setValue={setCaseSensitive}
            />
            <ToggleButton
              label={"Whole word"}
              icon={<WholeWord boxSize={5} />}
              value={wholeWordSearch}
              setValue={setWholeWordSearch}
            />
          </HStack>
        </InputRightElement>
      </InputGroup>

      <Box
        flexShrink={0}
        minWidth={16}
        flexDirection={"row"}
        display={"flex"}
        justifyContent={"center"}>
        {totalSearchResults > 0 ? (
          <Box marginLeft={2} marginRight={"1"} color={"text.tertiary"}>
            {activeSearchResult + 1} of {totalSearchResults}
          </Box>
        ) : searchString ? (
          <Box marginLeft={2} marginRight={"1"} color={"text.tertiary"}>
            0 of 0
          </Box>
        ) : (
          <></>
        )}
      </Box>

      <Tooltip label={"Previous search result"}>
        <IconButton
          variant={"ghost"}
          onClick={prevSearchResult}
          aria-label={"Previous search result"}
          disabled={!totalSearchResults}
          icon={<ArrowUp boxSize={5} />}
        />
      </Tooltip>

      <Tooltip label={"Next search result"}>
        <IconButton
          variant={"ghost"}
          onClick={nextSearchResult}
          aria-label={"Next search result"}
          disabled={!totalSearchResults}
          icon={<ArrowDown boxSize={5} />}
        />
      </Tooltip>
    </HStack>
  )
}

function ToggleButton({
  label,
  icon,
  value,
  setValue,
}: {
  label: string
  icon: ReactElement | undefined
  value: boolean
  setValue: (value: boolean) => void
}) {
  const hoverBgColor = useColorModeValue("gray.100", "gray.700")

  return (
    <Tooltip label={label}>
      <IconButton
        borderRadius={"100%"}
        variant={"ghost"}
        color={value ? "white" : undefined}
        backgroundColor={value ? "primary.400" : undefined}
        _hover={{
          bg: value ? "primary.600" : hoverBgColor,
        }}
        aria-label={label}
        fontFamily={"mono"}
        icon={icon}
        onClick={() => setValue(!value)}
      />
    </Tooltip>
  )
}
