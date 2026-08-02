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
        <NavLink to="/" className={styles.logo}>
          <img src="/icon.svg" alt="Itema" className={styles.logoIcon} width={32} height={32} />
          <span className={styles.logoText}>Itema</span>
        </NavLink>
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
