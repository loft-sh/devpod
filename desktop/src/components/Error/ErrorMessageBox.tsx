import { Box, Text, useColorModeValue } from "@chakra-ui/react"

type TErrorMessageBox = Readonly<{ error: Error }>
export function ErrorMessageBox({ error }: TErrorMessageBox) {
  const backgroundColor = useColorModeValue("red.100", "red.200")
  const textColor = useColorModeValue("red.700", "red.800")

  return (
    <Box
      backgroundColor={backgroundColor}
      marginTop="4"
      padding="4"
      borderRadius="md"
      display="inline-block">
      <Text color={textColor}>
        {error.message.split("\n").map((line) => (
          <>
            {line}
            <br />
          </>
        ))}
      </Text>
    </Box>
  )
}
