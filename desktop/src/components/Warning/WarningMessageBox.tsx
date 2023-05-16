import { Box, Text, useColorModeValue } from "@chakra-ui/react"

type TWarningMessageBox = Readonly<{ warning: string }>
export function WarningMessageBox({ warning }: TWarningMessageBox) {
  const backgroundColor = useColorModeValue("orange.100", "orange.200")
  const textColor = useColorModeValue("orange.700", "orange.800")

  return (
    <Box
      backgroundColor={backgroundColor}
      marginTop="4"
      padding="4"
      borderRadius="md"
      userSelect="auto"
      display="inline-block">
      <Text color={textColor}>{warning}</Text>
    </Box>
  )
}
