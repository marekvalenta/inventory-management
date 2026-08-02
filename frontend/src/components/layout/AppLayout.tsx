import { useMediaQuery } from '../../hooks/useMediaQuery'
import { MobileLayout } from './MobileLayout'
import { DesktopLayout } from './DesktopLayout'

export function AppLayout() {
  const isDesktop = useMediaQuery('(min-width: 1024px)')
  return isDesktop ? <DesktopLayout /> : <MobileLayout />
}
