import { useState, type KeyboardEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { MagnifyingGlassIcon } from '@radix-ui/react-icons'
import styles from './SearchBar.module.css'

export function SearchBar() {
  const [term, setTerm] = useState('')
  const navigate = useNavigate()

  const handleSubmit = () => {
    const q = term.trim()
    if (q.length > 0) {
      navigate(`/search?q=${encodeURIComponent(q)}`)
    }
  }

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      handleSubmit()
    }
  }

  return (
    <div className={styles.wrapper}>
      <div className={styles.inputWrapper}>
        <MagnifyingGlassIcon className={styles.icon} width={18} height={18} />
        <input
          type="search"
          className={styles.input}
          placeholder="Search inventory..."
          value={term}
          onChange={(e) => setTerm(e.target.value)}
          onKeyDown={handleKeyDown}
        />
      </div>
    </div>
  )
}
