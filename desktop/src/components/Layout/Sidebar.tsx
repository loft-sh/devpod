import { BellIcon } from "@chakra-ui/icons"
import {
  Box,
  BoxProps,
  Flex,
  Grid,
  IconButton,
  Link,
  Popover,
  PopoverArrow,
  PopoverBody,
  PopoverContent,
  PopoverHeader,
  PopoverTrigger,
  Portal,
  Spinner,
  useColorModeValue,
  useToken,
  VStack,
} from "@chakra-ui/react"
import { ReactElement, ReactNode } from "react"
import { LinkProps, NavLink as RouterLink } from "react-router-dom"
import { useWorkspaceActions } from "../../contexts"
import { DevpodIcon } from "../../icons"

type TSidebarProps = Readonly<{ children?: readonly ReactElement[] }> & BoxProps
export function Sidebar({ children, ...boxProps }: TSidebarProps) {
  const iconColor = useToken("colors", "primary")
  const sidebarBackgroundColor = useColorModeValue("gray.100", "gray.700")
  const actions = useWorkspaceActions()

  return (
    <Grid
      templateRows="6rem 1fr 6rem"
      width="full"
      backgroundColor={sidebarBackgroundColor}
      height="100vh"
      {...boxProps}>
      <Flex paddingLeft="6" alignItems="center" justify="flex-start" width="full">
        <DevpodIcon boxSize={8} color={iconColor} />
      </Flex>
      <VStack align="start">{children}</VStack>
      <VStack>
        <Popover placement="top">
          <PopoverTrigger>
            <Box width="41px" height="41px" position="relative">
              {actions.length > 0 && (
                <Spinner
                  color="primary"
                  width="41px"
                  height="41px"
                  position="absolute"
                  thickness="5px"
                  speed="750ms"
                />
              )}
              <IconButton
                top="3px"
                left="3px"
                variant="solid"
                backgroundColor="white"
                position="absolute"
                size="md"
                rounded="full"
                aria-label="Show onging operations"
                icon={<BellIcon />}
              />
            </Box>
          </PopoverTrigger>
          <Portal>
            <PopoverContent>
              <PopoverArrow />
              <PopoverHeader>Notifications</PopoverHeader>
              <PopoverBody>Content</PopoverBody>
            </PopoverContent>
          </Portal>
        </Popover>
      </VStack>
    </Grid>
  )
}

// {operations.map((operation) => (
//   <HStack key={operation.id[0]} width="full" paddingX={8} align="center">
//     <Text>
//       {operation.id[0]} {operation.context?.workspaceID ?? ""}
//     </Text>
//     <Progress size="xs" width="full" isIndeterminate />
//   </HStack>
//))}

type TSidebarMenuProps = Pick<LinkProps, "to"> & Readonly<{ children?: ReactNode }>
export function SidebarMenuItem({ to, children }: TSidebarMenuProps) {
  const activeBackgroundColor = useColorModeValue("gray.300", "gray.600")

  return (
    <Box paddingX="4" width="full">
      <Link
        display="block"
        paddingX="4"
        paddingY="4"
        as={RouterLink}
        to={to}
        width="full"
        borderRadius="sm"
        _activeLink={{ fontWeight: "bold", backgroundColor: activeBackgroundColor }}>
        {children}
      </Link>
    </Box>
  )
}
