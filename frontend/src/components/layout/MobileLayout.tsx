import { NavLink, Outlet } from 'react-router-dom'
import { HomeIcon, CubeIcon, BookmarkIcon, GearIcon } from '@radix-ui/react-icons'
import styles from './MobileLayout.module.css'

const navItems = [
  { to: '/locations', label: 'Locations', icon: HomeIcon },
  { to: '/definitions', label: 'Definitions', icon: CubeIcon },
  { to: '/tags', label: 'Tags', icon: BookmarkIcon },
  { to: '/settings', label: 'Settings', icon: GearIcon },
]

export function MobileLayout() {
  return (
    <div className={styles.shell}>
      <header className={styles.header}>
        <span className={styles.headerTitle}>Inventory</span>
        <div className={styles.headerActions} />
      </header>
      <main className={styles.content}>
        <Outlet />
      </main>
      <nav className={styles.bottomNav}>
        {navItems.map(({ to, label, icon: Icon }) => (
          <NavLink
            key={to}
            to={to}
            end={to === '/locations'}
            className={({ isActive }) =>
              `${styles.navTab} ${isActive ? styles.navTabActive : ''}`
            }
          >
            <Icon width={24} height={24} />
            <span>{label}</span>
          </NavLink>
        ))}
      </nav>
    </div>
  )
}
