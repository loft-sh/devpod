import { Box, BoxProps, forwardRef } from "@chakra-ui/react"

type TFormProps = Readonly<{
  ref?: React.Ref<HTMLFormElement>
  children: React.ReactNode
  onSubmit?: React.FormEventHandler<HTMLFormElement>
}> &
  Omit<BoxProps, "onSubmit">

export const Form = forwardRef<TFormProps, typeof Box>(function InnerForm(
  { children, onSubmit, ...boxProps },
  ref
) {
  return (
    <Box
      ref={ref}
      as="form"
      spellCheck={false}
      width="full"
      display="flex"
      flexFlow="column nowrap"
      {...boxProps}
      onSubmit={onSubmit}>
      {children}
    </Box>
  )
})
