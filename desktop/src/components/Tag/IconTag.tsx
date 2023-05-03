import { Tag, TagLabel, Tooltip, useColorModeValue } from "@chakra-ui/react"
import { cloneElement, ReactElement } from "react"

type TIconTagProps = Readonly<{
  icon: ReactElement
  label: string
  infoText?: string
}>

export function IconTag({ icon: iconProps, label, infoText }: TIconTagProps) {
  const tagColor = useColorModeValue("gray.700", "gray.300")
  const icon = cloneElement(iconProps, { boxSize: 4 })

  return (
    <Tooltip label={infoText}>
      <Tag borderRadius="full" color={tagColor}>
        {icon}
        <TagLabel marginLeft={2}>{label}</TagLabel>
      </Tag>
    </Tooltip>
  )
}
