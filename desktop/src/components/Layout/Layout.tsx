import {ReactNode} from "react";
import {Box, VStack} from "@chakra-ui/react";
import {Header} from "../Header/Header";

export function Layout({ children }: Readonly<{ children?: ReactNode }>) {
    return (
        <VStack spacing={4} height="100vh">
            <Header />
            <Box width="full" height="full" overflowY="auto">
                {children}
            </Box>
        </VStack>
    )
}
