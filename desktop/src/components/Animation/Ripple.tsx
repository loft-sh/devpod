import { Icon, IconProps } from "@chakra-ui/react"
import { motion } from "framer-motion"

const initial = {
  r: 4,
  opacity: 0.3,
}
const animate = { r: 12, opacity: 0 }
const transition = { duration: 4, repeat: Infinity }
export function Ripple(props: IconProps) {
  return (
    <Icon {...props} fill="currentColor" viewBox="0 0 24 24">
      <motion.circle cx="12" cy="12" initial={initial} animate={animate} transition={transition} />
      <motion.circle
        cx="12"
        cy="12"
        initial={initial}
        animate={animate}
        transition={{ ...transition, delay: 1 }}
      />
      <motion.circle
        cx="12"
        cy="12"
        initial={initial}
        animate={animate}
        transition={{ ...transition, delay: 2 }}
      />
    </Icon>
  )
}
