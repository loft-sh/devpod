import { Card, Image } from "@chakra-ui/react"
import { UseFormSetValue } from "react-hook-form"
import { FieldName, TFormValues } from "./types"

type TExampleCardProps = {
  image: string
  source: string
  currentSource: string

  setValue: UseFormSetValue<TFormValues>
}

export function ExampleCard(props: TExampleCardProps) {
  return (
    <Card
      _hover={{
        boxShadow: "rgba(186, 80, 255, 0.8) 0px 1px 4px 0px",
      }}
      transition={"box-shadow .5s"}
      width={"100px"}
      height={"100px"}
      alignItems={"center"}
      display={"flex"}
      justifyContent={"center"}
      cursor={"pointer"}
      border={props.currentSource === props.source ? "#BA50FF 1px solid" : undefined}
      onClick={() => {
        props.setValue(FieldName.SOURCE, props.currentSource === props.source ? "" : props.source, {
          shouldDirty: true,
        })
      }}
      padding={"10px"}>
      <Image objectFit="contain" overflow="hidden" width="fill" height="fill" src={props.image} />
    </Card>
  )
}
