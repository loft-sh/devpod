import {
  BoxProps,
  Card,
  Image,
  ImageProps,
  Text,
  Tooltip,
  forwardRef,
  useColorModeValue,
  useToken,
} from "@chakra-ui/react"
import { ComponentType, ReactElement, cloneElement, useMemo } from "react"

type TExampleCardProps = {
  name: string
  image?: string | ReactElement
  size?: keyof typeof sizes

  isSelected?: boolean
  isDisabled?: boolean
  showTooltip?: boolean
  onClick?: () => void
} & BoxProps

const sizes: Record<"sm" | "md" | "lg", BoxProps["width"]> = {
  sm: "12",
  md: "16",
  lg: "20",
} as const

export const ExampleCard = forwardRef<TExampleCardProps, ComponentType<typeof Card>>(
  function InnerExampleCard(
    {
      name,
      image,
      isSelected,
      isDisabled,
      size = "lg",
      showTooltip = true,
      onClick,
      ...restBoxProps
    },
    ref
  ) {
    const hoverBackgroundColor = useColorModeValue("gray.50", "gray.800")

    const primaryColorLight = useToken("colors", "primary.400")
    const primaryColorDark = useToken("colors", "primary.800")

    const selectedProps = isSelected
      ? {
          _before: {
            content: '""',
            position: "absolute",
            top: 0,
            bottom: 0,
            left: 0,
            right: 0,
            background: `linear-gradient(135deg, ${primaryColorLight}55 30%, ${primaryColorDark}55, ${primaryColorDark}88)`,
            opacity: 0.7,
            width: "full",
            height: "full",
          },
          boxShadow: `inset 0px 0px 0px 1px ${primaryColorDark}55`,
        }
      : {}

    const disabledProps = isDisabled ? { filter: "grayscale(100%)", cursor: "not-allowed" } : {}

    const imageElement = useMemo(() => {
      if (image === undefined) {
        return null
      }
      const imageProps: ImageProps = { objectFit: "fill", overflow: "hidden", zIndex: "1" }
      if (typeof image === "string") {
        return <Image src={image} {...imageProps} />
      }

      return cloneElement(image, imageProps)
    }, [image])

    return (
      <Tooltip textTransform={"capitalize"} label={name} isDisabled={size === "lg" || !showTooltip}>
        <Card
          ref={ref}
          {...restBoxProps}
          variant="unstyled"
          width={sizes[size]}
          height={sizes[size]}
          alignItems="center"
          display="flex"
          justifyContent="center"
          cursor="pointer"
          boxSizing="border-box"
          position="relative"
          backgroundColor="transparent"
          _dark={{
            backgroundColor: "transparent",
            _hover: {
              backgroundColor: isDisabled || isSelected ? undefined : hoverBackgroundColor,
            },
          }}
          overflow="hidden"
          _hover={{ backgroundColor: isDisabled || isSelected ? undefined : hoverBackgroundColor }}
          {...(onClick && !isDisabled && !isSelected ? { onClick } : {})}
          {...selectedProps}
          {...disabledProps}>
          {imageElement}
          {size === "lg" && (
            <Text
              paddingBottom="1"
              fontSize="11px"
              color="gray.500"
              _dark={{ color: "gray.300" }}
              fontWeight="medium"
              overflow="hidden"
              maxWidth={sizes[size]}
              textOverflow="ellipsis"
              whiteSpace="nowrap"
              textTransform={"capitalize"}>
              {name}
            </Text>
          )}
        </Card>
      </Tooltip>
    )
  }
)
