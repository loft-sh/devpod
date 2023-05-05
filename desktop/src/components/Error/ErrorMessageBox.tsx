import { Box, Text, useColorModeValue } from "@chakra-ui/react"
import React from "react"

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
      <Text color={textColor} fontFamily="monospace">
        {error.message.split("\n").map((line, index) => (
          <React.Fragment key={index}>
            {line}
            <br />
          </React.Fragment>
        ))}
      </Text>
    </Box>
  )
}
