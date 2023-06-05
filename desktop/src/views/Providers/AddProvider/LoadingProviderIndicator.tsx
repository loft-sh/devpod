import { Box, HStack, Text } from "@chakra-ui/react"

export function LoadingProviderIndicator({ label }: Readonly<{ label: string | undefined }>) {
  return (
    <HStack marginTop="2" justifyContent="center" alignItems="center" color="gray.600">
      <Text fontWeight="medium">{label}</Text>
      <Box as="svg" height="3" marginInlineStart="0 !important" width="8" viewBox="0 0 48 30">
        <circle fill="currentColor" stroke="none" cx="6" cy="24" r="6">
          <animateTransform
            attributeName="transform"
            dur="1s"
            type="translate"
            values="0 0; 0 -12; 0 0; 0 0; 0 0; 0 0"
            repeatCount="indefinite"
            begin="0"
          />
        </circle>
        <circle fill="currentColor" stroke="none" cx="24" cy="24" r="6">
          <animateTransform
            id="op"
            attributeName="transform"
            dur="1s"
            type="translate"
            values="0 0; 0 -12; 0 0; 0 0; 0 0; 0 0"
            repeatCount="indefinite"
            begin="0.3s"
          />
        </circle>
        <circle fill="currentColor" stroke="none" cx="42" cy="24" r="6">
          <animateTransform
            id="op"
            attributeName="transform"
            dur="1s"
            type="translate"
            values="0 0; 0 -12; 0 0; 0 0; 0 0; 0 0"
            repeatCount="indefinite"
            begin="0.6s"
          />
        </circle>
      </Box>
    </HStack>
  )
}
