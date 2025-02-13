import {
  Button,
  HStack,
  Input,
  Popover,
  PopoverArrow,
  PopoverContent,
  PopoverTrigger,
  Tab,
  TabList,
  TabPanels,
  Tabs,
  Tooltip,
  useToken,
  VStack,
} from "@chakra-ui/react"
import { ChevronDownIcon } from "@chakra-ui/icons"
import React, { ChangeEvent, useCallback, useMemo, useState } from "react"
import { ERevisionType, REVISION_TYPE_CONFIG } from "@/components/SourceInput/type"
import { useBorderColor } from "@/Theme"
import { extractRevisionType, extractSourceValue } from "@/components/SourceInput/url-parser"

export function RevisionPopover({
  disabled,
  source,
  onApplyRequested,
  triggerHeight,
}: {
  source: string
  disabled?: boolean
  onApplyRequested: (revision: string, revisionType: ERevisionType) => void
  triggerHeight?: React.ComponentProps<typeof Button>["height"]
}) {
  const borderColor = useBorderColor()
  const errorBorderColor = useToken("colors", "red.500")

  const [open, setOpen] = useState<boolean>(false)
  const [revisionType, setRevisionType] = useState<ERevisionType>(ERevisionType.BRANCH)
  const [revision, setRevision] = useState<string>("")
  const [revisionTouched, setRevisionTouched] = useState<boolean>(false)

  const revisionConfig = REVISION_TYPE_CONFIG[revisionType]

  const revisionRegex = useMemo(() => {
    return new RegExp(`^${revisionConfig.partialRegex.source}`)
  }, [revisionConfig])

  const tab = useMemo(() => {
    return Object.values(ERevisionType).findIndex((t) => t === revisionType)
  }, [revisionType])

  const changeRevision = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => {
      setRevision(e.target.value)
      if (!revisionTouched) {
        setRevisionTouched(true)
      }
    },
    [setRevision, revisionTouched, setRevisionTouched]
  )

  const changeRevisionType = useCallback(
    (index: number) => {
      const typeItems = Object.values(ERevisionType)
      const revisionType = typeItems[index]!

      setRevisionType(revisionType)
      setRevision("")
      setRevisionTouched(false)
    },
    [setRevision, setRevisionType, setRevisionTouched]
  )

  const revisionValid = useMemo(() => {
    const formatted = revisionConfig.formatter(revision)

    return revisionRegex.test(formatted)
  }, [revision, revisionConfig, revisionRegex])

  const onClose = useCallback(() => {
    setOpen(false)
  }, [setOpen])

  const onOpen = useCallback(() => {
    setOpen(true)

    const initialRevisionType = extractRevisionType(source) ?? ERevisionType.BRANCH
    const sourceValue = extractSourceValue(source, initialRevisionType)

    setRevisionType(initialRevisionType)
    setRevision(sourceValue.revision ?? "")
    setRevisionTouched(!!sourceValue.revision)
  }, [setOpen, source, setRevisionType, setRevisionTouched])

  const apply = useCallback(() => {
    setOpen(false)
    onApplyRequested(revision, revisionType)
  }, [setOpen, onApplyRequested, revision, revisionType])

  return (
    <Popover isLazy onOpen={onOpen} onClose={onClose} isOpen={open}>
      <PopoverTrigger>
        <Button
          _invalid={{
            borderStyle: "solid",
            borderWidth: "1px",
            borderLeftWidth: 0,
            borderColor: errorBorderColor,
          }}
          aria-invalid={disabled ? "true" : undefined}
          isDisabled={disabled}
          leftIcon={<ChevronDownIcon boxSize={5} />}
          transform="auto"
          borderTopLeftRadius={0}
          borderBottomLeftRadius={0}
          borderTopWidth={"thin"}
          borderRightWidth={"thin"}
          borderBottomWidth={"thin"}
          borderColor={borderColor}
          minW="32"
          height={triggerHeight ?? "10"}>
          Advanced...
        </Button>
      </PopoverTrigger>
      <PopoverContent width="auto" padding="4">
        <PopoverArrow />
        <VStack>
          <Tabs variant="muted" size="sm" index={tab} onChange={changeRevisionType}>
            <TabList>
              <Tab>Branch</Tab>
              <Tab>Commit</Tab>
              <Tab>Pull Request</Tab>
              <Tab>Sub Folder</Tab>
            </TabList>
            <TabPanels paddingTop="2">
              <Tooltip
                placement="top-start"
                label={
                  !revisionValid && revisionTouched ? `${revisionType} is malformed` : undefined
                }>
                <Input
                  value={revision}
                  aria-invalid={!revisionValid && revisionTouched ? "true" : undefined}
                  isInvalid={!revisionValid && revisionTouched}
                  _invalid={{
                    borderStyle: "solid",
                    borderWidth: "1px",
                    borderColor: errorBorderColor,
                  }}
                  onChange={changeRevision}
                  placeholder={revisionConfig.placeholder}
                />
              </Tooltip>
            </TabPanels>
          </Tabs>
        </VStack>

        <HStack mt={4} alignItems={"center"} justify={"end"} gap={2}>
          <Button variant={"outlined"} onClick={onClose}>
            Cancel
          </Button>
          <Button isDisabled={!revisionValid} onClick={apply}>
            Apply
          </Button>
        </HStack>
      </PopoverContent>
    </Popover>
  )
}
