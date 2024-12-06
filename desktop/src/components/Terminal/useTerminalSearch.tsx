import React, { useCallback, useEffect, useRef, useState } from "react"
import { TTerminal } from "@/components"

export type TSearchOptions = {
  searchString?: string
  caseSensitive?: boolean
  wholeWordSearch?: boolean
}

type TSearchResult = [row: number, col: number, len: number]

type TDisplayLine = {
  index: number
  text: string
  inputLine: number
  startCol: number
  endCol: number
}

type THighlight = {
  displayRow: number

  // Bounds of the highlight within a row.
  startCol: number
  endCol: number

  // Which search result this highlight is attached to.
  resultIndex: number
}

type TDisplayLineMap = { [key: number]: TDisplayLine[] }

// Used for keeping track of anchors/jump points in the wrapped lines to jump to for any given search result.
type TJumpMap = { [resultIndex: number]: number }

type TSearchState = {
  searchableLines: string[]
  disposables: VoidFunction[]
  searchOptions: TSearchOptions | undefined
  jumpMap: TJumpMap
  debounce: number | undefined
  activeSearchResult: number
  preWrappedLines: TDisplayLineMap | undefined
  searchResults: TSearchResult[]
  highlights: THighlight[]
}

export function useTerminalSearch(
  terminalRef: React.MutableRefObject<TTerminal | null>,
  searchOptions?: TSearchOptions
) {
  const [totalSearchResults, setTotalSearchResults] = useState<number>(-1)
  const [activeSearchResult, setActiveSearchResult] = useState<number>(0)

  // We have to exercise caution not to re-generate connectStream,
  // so we have to store a lot of state outside the usual mechanisms.
  // Otherwise, we will make the terminal flicker.

  const searchStateRef = useRef<TSearchState>({
    searchableLines: [],
    disposables: [],
    searchOptions,
    jumpMap: {},
    debounce: undefined,
    activeSearchResult: 0,
    preWrappedLines: undefined,
    searchResults: [],
    highlights: [],
  })

  const clearDisposables = useCallback(() => {
    const toClear = searchStateRef.current.disposables
    toClear.forEach((disposable) => disposable())
    searchStateRef.current.disposables = []
  }, [searchStateRef])

  const repaintHighlights = useCallback(
    (highlights: THighlight[]) => {
      const terminal = terminalRef.current
      const displayLines = searchStateRef.current.preWrappedLines
      const searchResults = searchStateRef.current.searchResults

      if (!displayLines || !terminal || !searchResults.length) {
        return
      }

      searchStateRef.current.highlights = highlights

      for (const highlight of highlights) {
        const isActive = highlight.resultIndex === searchStateRef.current.activeSearchResult

        const disposable = terminal.highlight(
          highlight.displayRow,
          highlight.startCol,
          highlight.endCol - highlight.startCol,
          isActive ? "#E4ADFF" : "#8E00EB",
          isActive
        )

        if (disposable) {
          searchStateRef.current.disposables.push(disposable)
        }
      }
    },
    [searchStateRef, terminalRef]
  )

  // When the terminal is resized, we need to re-calculate the highlights and jump anchors,
  // as these have to be positioned per wrapped line.
  const onResize = useCallback(
    (cols: number) => {
      const displayLines = wrapLines(searchStateRef.current.searchableLines, cols)
      searchStateRef.current.preWrappedLines = displayLines

      const [highlights, jumpMap] = generateHighlights(
        searchStateRef.current.searchResults,
        displayLines
      )

      searchStateRef.current.jumpMap = jumpMap

      clearDisposables()
      repaintHighlights(highlights)
    },
    [repaintHighlights, clearDisposables]
  )

  // Currently we kind of have to split the state for the active search result to allow re-rendering
  // but also to prevent a change of the connectStream function, so this is a setter that handles both ends.
  const changeActiveSearchResult = useCallback(
    (result: number, repaint?: boolean) => {
      searchStateRef.current.activeSearchResult = result
      setActiveSearchResult(result)
      if (repaint) {
        clearDisposables()
        repaintHighlights(searchStateRef.current.highlights)
      }
    },
    [searchStateRef, repaintHighlights, clearDisposables, setActiveSearchResult]
  )

  const jumpToResult = useCallback(
    (resultIndex: number, repaint?: boolean) => {
      changeActiveSearchResult(resultIndex, repaint)
      const jumpIndex = searchStateRef.current.jumpMap[resultIndex]
      if (jumpIndex != null) {
        terminalRef.current?.getTerminal()?.scrollToLine(jumpIndex)
      }
    },
    [terminalRef, searchStateRef, changeActiveSearchResult]
  )

  const jumpNext = useCallback(() => {
    if (totalSearchResults <= 1) {
      return
    }

    const nextIndex = wrapNumber(searchStateRef.current.activeSearchResult + 1, totalSearchResults)
    jumpToResult(nextIndex, true)
  }, [totalSearchResults, jumpToResult, searchStateRef])

  const jumpPrev = useCallback(() => {
    if (totalSearchResults <= 1) {
      return
    }

    const prevIndex = wrapNumber(searchStateRef.current.activeSearchResult - 1, totalSearchResults)

    jumpToResult(prevIndex, true)
  }, [totalSearchResults, jumpToResult, searchStateRef])

  const performSearch = useCallback(
    (opts: TSearchOptions | undefined, jump?: boolean) => {
      clearDisposables()

      const terminal = terminalRef.current
      const inputLines = searchStateRef.current.searchableLines

      if (!terminal || !opts?.searchString || !inputLines.length) {
        setTotalSearchResults(-1)
        searchStateRef.current.highlights = []
        searchStateRef.current.searchResults = []

        return
      }

      const results = (searchStateRef.current.searchResults = search(inputLines, opts))
      setTotalSearchResults(results.length)

      if (!results.length) {
        searchStateRef.current.highlights = []
        searchStateRef.current.searchResults = []

        return
      }

      // We need to calculate wrapped lines:
      // xterm internally treats lines that get wrapped as seperate lines & uses these for navigation.
      // Did not find a reasonable way to extract these from xterm, so we do it ourselves.
      let displayLines = searchStateRef.current.preWrappedLines

      // Optimization: Don't calculate the wrapping if we've done it before for the current set of lines.
      // Requires the consumer of this API to properly reset the preWrappedLines when it feeds in new lines.
      if (!displayLines) {
        searchStateRef.current.preWrappedLines = displayLines = wrapLines(
          inputLines,
          terminal.getTerminal()?.cols ?? 0
        )
      }

      searchStateRef.current.preWrappedLines = displayLines

      const [highlights, jumpMap] = generateHighlights(results, displayLines)

      searchStateRef.current.jumpMap = jumpMap

      if (jump) {
        jumpToResult(0)
      }

      repaintHighlights(highlights)
    },
    [searchStateRef, terminalRef, jumpToResult, clearDisposables, repaintHighlights]
  )

  // This is used for a search while the terminal still gets new input.
  // Since it is likely that there are going to be many new lines within a short timeframe,
  // we debounce the search on new input so we don't calculate results uselessly.
  const debounceSearchResults = useCallback(
    (opts: TSearchOptions | undefined) => {
      if (searchStateRef.current.debounce != null) {
        clearTimeout(searchStateRef.current.debounce)
      }

      const timeout = setTimeout(() => {
        if (searchStateRef.current.debounce === timeout) {
          searchStateRef.current.debounce = undefined
        }
        performSearch(opts)
      }, 100) as unknown as number

      searchStateRef.current.debounce = timeout
    },
    [performSearch, searchStateRef]
  )

  const resetSearch = useCallback(() => {
    // Remove all dangling highlights.
    clearDisposables()

    // Reset internal search state.
    searchStateRef.current = {
      // We keep the current search options because these are synced from the outside.
      searchOptions: searchStateRef.current.searchOptions,
      searchableLines: [],
      disposables: [],
      jumpMap: {},
      debounce: undefined,
      activeSearchResult: 0,
      preWrappedLines: undefined,
      searchResults: [],
      highlights: [],
    }

    // Reset externally available state.
    changeActiveSearchResult(0)
    setTotalSearchResults(-1)
  }, [setTotalSearchResults, changeActiveSearchResult, searchStateRef, clearDisposables])

  // Synchronize the search options with the consuming components and re-trigger the search when they change.
  useEffect(() => {
    searchStateRef.current.searchOptions = searchOptions

    performSearch(searchOptions, true)
  }, [searchOptions, performSearch])

  return {
    internals: {
      searchStateRef,
      debounceSearchResults,
      resetSearch,
      onResize,
    },
    searchApi: {
      nextSearchResult: jumpNext,
      prevSearchResult: jumpPrev,
      totalSearchResults,
      activeSearchResult,
    },
  }
}

function search(inputLines: string[], opts: TSearchOptions) {
  let pattern = escapeRegex(opts.searchString!)

  if (opts.wholeWordSearch) {
    pattern = `\\b${pattern}\\b`
  }

  return inputLines.reduce((accum, line, index) => {
    const regex = new RegExp(pattern, opts.caseSensitive ? "g" : "ig")

    let match

    while ((match = regex.exec(line)) && match[0]) {
      accum.push([index, match.index, match[0].length])
    }

    return accum
  }, [] as TSearchResult[])
}

/**
 * Generates one or more highlight specification per search result.
 * One search result can produce multiple highlights as they can get wrapped and we have to instantiate
 * decorations per wrapped line.
 */
function generateHighlights(
  results: TSearchResult[],
  displayLinesMap: TDisplayLineMap
): [THighlight[], TJumpMap] {
  const highlights: THighlight[] = []
  const jumpMap: TJumpMap = {}

  results.forEach((result, resultIndex) => {
    const [inputLine, start, length] = result
    const end = start + length
    const displayLines = displayLinesMap[inputLine]!

    displayLines.forEach((displayLine) => {
      const overlapStart = Math.max(start, displayLine.startCol)
      const overlapEnd = Math.min(end, displayLine.endCol)

      if (overlapStart < overlapEnd) {
        if (jumpMap[resultIndex] == null) {
          jumpMap[resultIndex] = displayLine.index
        }

        highlights.push({
          displayRow: displayLine.index,
          startCol: overlapStart - displayLine.startCol,
          endCol: overlapEnd - displayLine.startCol,
          resultIndex,
        })
      }
    })
  })

  return [highlights, jumpMap]
}

/**
 * Emulates the wrapping of lines by max columns as it is done in the terminal.
 */
function wrapLines(inputLines: string[], cols: number) {
  const result: TDisplayLineMap = {}

  let count = 0

  for (let inputLineIndex = 0; inputLineIndex < inputLines.length; inputLineIndex++) {
    const line = inputLines[inputLineIndex]!

    let startCharIndex = 0

    const displayLines: TDisplayLine[] = []

    // Special case: There may be lines with 0 length.
    // While no displayLine is required for this, we still need to increment the counter.
    if (line.length === 0) {
      count++
    } else {
      while (startCharIndex < line.length) {
        const endCharIndex = Math.min(startCharIndex + cols, line.length)
        const text = line.substring(startCharIndex, endCharIndex)

        displayLines.push({
          index: count++,
          text: text,
          inputLine: inputLineIndex,
          startCol: startCharIndex,
          endCol: endCharIndex,
        })

        startCharIndex = endCharIndex
      }
    }

    result[inputLineIndex] = displayLines
  }

  return result
}

/**
 * Wraps around numbers in the bounds [0-max].
 */
function wrapNumber(num: number, max: number) {
  return ((num % max) + max) % max
}

/**
 * Escapes anything that could be interpreted as regex syntax when parsing the string as a regex.
 */
function escapeRegex(str: string) {
  return str.replace(/[/\-\\^$*+?.()|[\]{}]/g, "\\$&")
}
