import React from "react"
import { Card, Image } from "@chakra-ui/react"
import { UseFormSetValue } from "react-hook-form"
import { FieldName, TFormValues } from "./types"

type TExampleCardProps = {
  image?: string
  setValue?: UseFormSetValue<TFormValues>
  source?: string
  currentSource: string

  selected?: boolean
  imageNode?: React.ReactNode
  onClick?: () => void
}

export function RecommendedProviderCard(props: TExampleCardProps) {
  return (
    <Card
      _hover={{
        boxShadow: "rgba(186, 80, 255, 0.8) 0px 1px 4px 0px",
      }}
      transition={"box-shadow .5s"}
      width={"120px"}
      height={"120px"}
      alignItems={"center"}
      display={"flex"}
      justifyContent={"center"}
      cursor={"pointer"}
      border={
        props.selected || props.currentSource === props.source ? "#BA50FF 1px solid" : undefined
      }
      onClick={
        props.onClick
          ? props.onClick
          : () => {
              props.setValue?.(
                FieldName.PROVIDER_SOURCE,
                props.currentSource === props.source ? "" : props.source!,
                {
                  shouldDirty: true,
                }
              )
            }
      }
      padding={"10px"}>
      {props.imageNode ? (
        props.imageNode
      ) : (
        <Image
          objectFit="cover"
          maxH={{ base: "100%", sm: "100px" }}
          maxW={{ base: "100%", sm: "100px" }}
          src={props.image}
        />
      )}
    </Card>
  )
}
