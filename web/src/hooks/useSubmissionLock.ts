import { useCallback, useRef, useState } from 'react'

function isFormValidationError(caught: unknown) {
  return typeof caught === 'object'
    && caught !== null
    && 'errorFields' in caught
    && Array.isArray(caught.errorFields)
}

export function useSubmissionLock() {
  const lockedRef = useRef(false)
  const [locked, setLocked] = useState(false)

  const runLocked = useCallback(async (task: () => void | Promise<void>) => {
    if (lockedRef.current) return

    lockedRef.current = true
    setLocked(true)
    try {
      await task()
    } catch (caught: unknown) {
      if (!isFormValidationError(caught)) throw caught
    } finally {
      lockedRef.current = false
      setLocked(false)
    }
  }, [])

  return [locked, runLocked] as const
}
