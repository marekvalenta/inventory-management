import { NavLink, Outlet } from 'react-router-dom'
import { HomeIcon, CubeIcon, BookmarkIcon, GearIcon } from '@radix-ui/react-icons'
import styles from './DesktopLayout.module.css'

const primaryNav = [
  { to: '/locations', label: 'Locations', icon: HomeIcon },
  { to: '/definitions', label: 'Definitions', icon: CubeIcon },
  { to: '/tags', label: 'Tags', icon: BookmarkIcon },
]

const secondaryNav = [
  { to: '/settings', label: 'Settings', icon: GearIcon },
]

export function DesktopLayout() {
  return (
    <div className={styles.shell}>
      <aside className={styles.sidebar}>
        <div className={styles.logo}>
          <span className={styles.logoMark}>INV</span>
          <span className={styles.logoText}>Inventory</span>
        </div>
        {primaryNav.map(({ to, label, icon: Icon }) => (
          <NavLink
            key={to}
            to={to}
            end={to === '/locations'}
            className={({ isActive }) =>
              `${styles.navItem} ${isActive ? styles.navItemActive : ''}`
            }
          >
            <Icon width={24} height={24} />
            <span>{label}</span>
          </NavLink>
        ))}
        <div className={styles.divider} />
        {secondaryNav.map(({ to, label, icon: Icon }) => (
          <NavLink
            key={to}
            to={to}
            className={({ isActive }) =>
              `${styles.navItem} ${isActive ? styles.navItemActive : ''}`
            }
          >
            <Icon width={24} height={24} />
            <span>{label}</span>
          </NavLink>
        ))}
      </aside>
      <div className={styles.main}>
        <Outlet />
      </div>
    </div>
  )
}
