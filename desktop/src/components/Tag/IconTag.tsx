import { ButtonProps, Tag, TagLabel, TagProps, Tooltip } from "@chakra-ui/react"
import { ReactElement, ReactNode, cloneElement } from "react"

type TIconTagProps = Readonly<{
  icon: ReactElement
  label: string
  info?: ReactNode
}> &
  Pick<ButtonProps, "onClick"> &
  TagProps

export function IconTag({ icon: iconProps, label, info, onClick, ...tagProps }: TIconTagProps) {
  const icon = cloneElement(iconProps, { boxSize: 4 })

  return (
    <Tooltip label={info}>
      <Tag
        borderRadius="full"
        onClick={onClick}
        role={onClick ? "button" : "status"}
        cursor={onClick ? "pointer" : "default"}
        {...tagProps}>
        {icon}
        <TagLabel marginLeft={2}>{label}</TagLabel>
      </Tag>
    </Tooltip>
  )
}
