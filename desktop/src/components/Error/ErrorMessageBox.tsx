import { Box, BoxProps, Text, useColorModeValue } from "@chakra-ui/react"
import React from "react"

type TErrorMessageBox = Readonly<{ error: Error }> & BoxProps
export function ErrorMessageBox({ error, ...boxProps }: TErrorMessageBox) {
  const backgroundColor = useColorModeValue("red.50", "red.100")
  const textColor = useColorModeValue("red.700", "red.800")

  return (
    <Box
      backgroundColor={backgroundColor}
      paddingY="1"
      paddingX="2"
      borderRadius="md"
      display="inline-block"
      {...boxProps}>
      <Text userSelect="text" cursor="text" color={textColor} fontFamily="monospace">
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
