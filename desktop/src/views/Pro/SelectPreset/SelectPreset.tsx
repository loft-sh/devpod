import {
  Box,
  Button,
  Grid,
  Heading,
  HStack,
  IconButton,
  Input,
  InputGroup,
  InputLeftElement,
  InputRightElement,
  Link,
  Spinner,
  Text,
  useToken,
  VStack,
} from "@chakra-ui/react"
import React, { ChangeEvent, useCallback, useMemo, useRef, useState } from "react"
import { useNavigate } from "react-router"
import { useProContext, useTemplates } from "@/contexts"
import { Routes } from "@/routes"
import { SearchIcon } from "@chakra-ui/icons"
import { BackToWorkspaces } from "@/views/Pro/BackToWorkspaces"
import { AiOutlineCloseCircle } from "react-icons/ai"
import { presetDisplayName } from "@/views/Pro/helpers"

export function SelectPreset() {
  const gridChildWidth = useToken("sizes", "96")
  const gridChildHeight = useToken("sizes", "48")

  const [searchString, setSearchString] = useState<string | undefined>(undefined)
  const searchInputRef = useRef<HTMLInputElement | null>(null)

  const { host } = useProContext()
  const { data: templates, isLoading: isTemplatesLoading } = useTemplates()

  const filteredPresets = useMemo(() => {
    return (templates?.presets ?? []).filter((preset) => {
      if (!searchString) {
        return true
      }

      if ((presetDisplayName(preset) ?? "").includes(searchString)) {
        return true
      }

      if (preset.spec?.source.image && preset.spec.source.image.includes(searchString)) {
        return true
      }

      return preset.spec?.source.git && preset.spec.source.git.includes(searchString)
    })
  }, [templates?.presets, searchString])

  const navigate = useNavigate()

  const createPlain = useCallback(() => {
    navigate(Routes.toProWorkspaceCreate(host))
  }, [navigate, host])

  const changeSearchString = useCallback((e: ChangeEvent<HTMLInputElement>) => {
    setSearchString(e.target.value ? e.target.value : undefined)
  }, [])

  return (
    <Box display={"flex"} flexFlow={"column"} w={"full"} mb="20">
      <BackToWorkspaces />
      <HStack w={"full"} align={"center"} justifyContent={"space-between"} mt={"4"} mb={"8"}>
        <VStack w={"full"} align={"start"} justify={"start"}>
          <Heading fontWeight={"thin"}>Create Workspace</Heading>
          <Text>
            Select a preset below or <Link onClick={createPlain}>create a custom workspace</Link>
          </Text>
        </VStack>
        <InputGroup>
          <InputLeftElement cursor={"text"} onClick={() => searchInputRef.current?.focus()}>
            <SearchIcon />
          </InputLeftElement>
          <Input
            ref={searchInputRef}
            value={searchString ?? ""}
            placeholder={"Filter by name, repo or image"}
            spellCheck={false}
            onChange={changeSearchString}
            bg={"white"}
          />
          {searchString && (
            <InputRightElement>
              <IconButton
                onClick={() => setSearchString(undefined)}
                aria-label={"Clear search"}
                variant={"ghost"}
                icon={<AiOutlineCloseCircle />}
              />
            </InputRightElement>
          )}
        </InputGroup>
      </HStack>
      {isTemplatesLoading ? (
        <Spinner />
      ) : (
        <Grid
          gridTemplateColumns={`repeat(auto-fit, ${gridChildWidth})`}
          rowGap={"5"}
          columnGap={"5"}
          w={"full"}>
          {!searchString && (
            <Box
              onClick={createPlain}
              display={"flex"}
              cursor={"pointer"}
              flexDir={"column"}
              alignItems={"center"}
              justifyContent={"center"}
              h={gridChildHeight}
              border={"1px"}
              borderColor={"divider.main"}
              boxSizing={"border-box"}
              borderRadius={"4px"}>
              <Button variant={"outline"} colorScheme={"primary"}>
                New Custom Workspace
              </Button>
            </Box>
          )}
          {filteredPresets.map((preset) => (
            <PresetBox
              key={preset.metadata!.name!}
              preset={preset.metadata?.name ?? ""}
              host={host}
              height={gridChildHeight}
              name={presetDisplayName(preset) ?? ""}
              source={preset.spec?.source.image ?? preset.spec?.source.git ?? ""}
            />
          ))}
        </Grid>
      )}
    </Box>
  )
}

function PresetBox({
  height,
  name,
  source,
  host,
  preset,
}: {
  height: string
  name: string
  source: string
  preset: string
  host: string
}) {
  const navigate = useNavigate()

  const createFromPreset = useCallback(() => {
    navigate(Routes.toProWorkspaceCreate(host, preset))
  }, [navigate, host, preset])

  return (
    <Box
      _hover={{
        borderColor: "primary.500",
      }}
      onClick={createFromPreset}
      h={height}
      cursor={"pointer"}
      display={"flex"}
      flexDir={"column"}
      justifyContent={"space-between"}
      bg={"white"}
      border={"1px"}
      borderColor={"divider.main"}
      boxSizing={"border-box"}
      boxShadow={"0px 8px 16px 4px rgba(0, 0, 0, 0.10)"}
      paddingY={"18px"}
      paddingX={"24px"}
      transitionProperty={"border-color"}
      transitionDuration={"0.3s"}
      borderRadius={"4px"}>
      <Box display={"flex"} flexDir={"column"}>
        <Heading fontSize={"md"} fontWeight={"semibold"} as={"h3"}>
          {name}
        </Heading>
        <Box display={"flex"} mt={"10px"} flexDir={"column"}>
          <Box fontSize={"sm"}>Source Code:</Box>
          <Box fontSize={"md"} color={"text.tertiary"}>
            {source}
          </Box>
        </Box>
      </Box>
      <Button flexShrink={0} variant={"primary"}>
        Select Preset
      </Button>
    </Box>
  )
}
