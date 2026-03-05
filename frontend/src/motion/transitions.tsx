import { motion } from "framer-motion"

type Props = {
  children: React.ReactNode
  className?: string
  testId?: string
  delay?: number
}

export function MotionSection({ children, className, testId, delay = 0 }: Props) {
  return (
    <motion.section
      data-testid={testId}
      className={className}
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.18, ease: "easeOut", delay }}
    >
      {children}
    </motion.section>
  )
}
