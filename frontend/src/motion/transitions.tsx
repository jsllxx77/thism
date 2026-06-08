import { motion, useReducedMotion, type Transition } from "framer-motion"

type Props = {
  children: React.ReactNode
  className?: string
  testId?: string
  delay?: number
  variant?: "page" | "section" | "content"
}

const motionPresets = {
  page: { y: 10, duration: 0.2 },
  section: { y: 8, duration: 0.18 },
  content: { y: 6, duration: 0.16 },
} as const

const easeOut: Transition["ease"] = [0.22, 1, 0.36, 1]

export function MotionSection({ children, className, testId, delay = 0, variant = "section" }: Props) {
  const reduceMotion = useReducedMotion()
  const preset = motionPresets[variant]

  return (
    <motion.section
      data-testid={testId}
      className={className}
      initial={reduceMotion ? false : { opacity: 0, y: preset.y }}
      animate={{ opacity: 1, y: 0 }}
      transition={reduceMotion ? { duration: 0 } : { duration: preset.duration, ease: easeOut, delay }}
    >
      {children}
    </motion.section>
  )
}
