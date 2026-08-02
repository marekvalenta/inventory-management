import { useState, useEffect, useCallback } from 'react'

export function useMediaQuery(query: string): boolean {
  const [matches, setMatches] = useState(() => {
    if (typeof window === 'undefined') return false
    return window.matchMedia(query).matches
  })

  const handleChange = useCallback((e: MediaQueryListEvent) => {
    setMatches(e.matches)
  }, [])

  useEffect(() => {
    const mql = window.matchMedia(query)
    mql.addEventListener('change', handleChange)
    return () => mql.removeEventListener('change', handleChange)
  }, [query, handleChange])

  return matches
}
