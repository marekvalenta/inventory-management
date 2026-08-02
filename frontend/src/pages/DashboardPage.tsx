import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import {
  HomeIcon,
  CubeIcon,
  LayersIcon,
  StackIcon,
  ChevronRightIcon,
  ChevronDownIcon,
  PlusIcon,
} from '@radix-ui/react-icons'
import { fetchDashboard, type DashboardData } from '../api/dashboard'
import { SearchBar } from '../components/common/SearchBar'
import styles from './DashboardPage.module.css'

export function DashboardPage() {
  const { data, isLoading, isError, refetch } = useQuery<DashboardData>({
    queryKey: ['dashboard'],
    queryFn: fetchDashboard,
    staleTime: 30_000,
  })

  if (isLoading) {
    return (
      <div className={styles.page}>
        <h1 className={styles.heading}>Itema</h1>
        <DashboardSkeleton />
      </div>
    )
  }

  if (isError || !data) {
    return (
      <div className={styles.page}>
        <h1 className={styles.heading}>Itema</h1>
        <div className={styles.errorState}>
          <p className={styles.errorTitle}>Something went wrong</p>
          <p className={styles.errorSubtitle}>Could not load dashboard.</p>
          <button className={styles.retryButton} onClick={() => refetch()}>
            Retry
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className={styles.page}>
      <h1 className={styles.heading}>Itema</h1>
      <SearchBar />
      {data.is_onboarding && <OnboardingGuide />}
      <StatCards stats={data.stats} isOnboarding={data.is_onboarding} />
      {!data.is_onboarding && <LocationBreakdown locations={data.locations} />}
    </div>
  )
}

function StatCards({ stats, isOnboarding }: { stats: DashboardData['stats']; isOnboarding: boolean }) {
  const cards = [
    {
      label: 'Locations',
      value: stats.locations_count,
      icon: HomeIcon,
      to: '/locations',
    },
    {
      label: 'Definitions',
      value: stats.definitions_count,
      icon: CubeIcon,
      to: '/definitions',
    },
    {
      label: 'Instances',
      value: stats.instances_count,
      icon: LayersIcon,
      to: '/locations',
    },
    {
      label: 'Total items',
      value: stats.total_quantity,
      icon: StackIcon,
      to: '/locations',
    },
  ]

  return (
    <div className={styles.statGrid}>
      {cards.map((card) => {
        const content = (
          <div className={styles.statCard}>
            <div className={styles.statHeader}>
              <card.icon className={styles.statIcon} width={20} height={20} />
              <span className={styles.statLabel}>{card.label}</span>
            </div>
            <span className={styles.statValue}>{card.value.toLocaleString()}</span>
          </div>
        )

        if (card.value > 0) {
          return (
            <Link key={card.label} to={card.to} className={styles.statLink}>
              {content}
            </Link>
          )
        }

        return (
          <div key={card.label} className={styles.statCardMuted}>
            {content}
          </div>
        )
      })}
    </div>
  )
}

function LocationBreakdown({ locations }: { locations: DashboardData['locations'] }) {
  if (locations.length === 0) {
    return (
      <div className={styles.emptyBreakdown}>
        <p className={styles.emptyTitle}>No locations added yet</p>
        <Link to="/locations" className={styles.emptyLink}>
          Go to locations
        </Link>
      </div>
    )
  }

  return (
    <div className={styles.breakdownSection}>
      <h2 className={styles.breakdownHeading}>Locations ({locations.length})</h2>
      <div className={styles.breakdownList}>
        {locations.map((loc) => (
          <LocationRow key={loc.id} node={loc} depth={0} />
        ))}
      </div>
    </div>
  )
}

function LocationRow({
  node,
  depth,
}: {
  node: DashboardData['locations'][number]
  depth: number
}) {
  const [expanded, setExpanded] = useState(false)

  return (
    <>
      <div
        className={styles.row}
        style={{ paddingLeft: `calc(var(--space-lg) + ${depth * 20}px)` }}
      >
        {node.children.length > 0 && (
          <button
            className={styles.expandButton}
            onClick={() => setExpanded(!expanded)}
            aria-label={expanded ? 'Collapse' : 'Expand'}
          >
            {expanded ? (
              <ChevronDownIcon width={18} height={18} />
            ) : (
              <ChevronRightIcon width={18} height={18} />
            )}
          </button>
        )}
        {node.children.length === 0 && <span className={styles.expandSpacer} />}
        <Link to={`/locations/${node.id}`} className={styles.rowLink}>
          <HomeIcon className={styles.rowIcon} width={16} height={16} />
          <span className={styles.rowName}>{node.name}</span>
        </Link>
        <span className={styles.rowBadge}>{node.instance_count.toLocaleString()}</span>
      </div>
      {expanded &&
        node.children.map((child) => (
          <LocationRow key={child.id} node={child} depth={depth + 1} />
        ))}
    </>
  )
}

function OnboardingGuide() {
  return (
    <div className={styles.onboardingCard}>
      <h3 className={styles.onboardingTitle}>Get Started</h3>
      <div className={styles.onboardingSteps}>
        <div className={styles.onboardingStep}>
          <span className={styles.stepNumber}>1</span>
          <div className={styles.stepContent}>
            <p className={styles.stepTitle}>Add locations</p>
            <p className={styles.stepDesc}>
              Rooms, shelves, boxes — set up your storage spaces
            </p>
            <Link to="/locations" className={styles.stepCta}>
              <PlusIcon width={14} height={14} />
              Add first location
            </Link>
          </div>
        </div>
        <div className={styles.onboardingStep}>
          <span className={styles.stepNumber}>2</span>
          <div className={styles.stepContent}>
            <p className={styles.stepTitle}>Define your items</p>
            <p className={styles.stepDesc}>
              WHAT you track — like "M3 Screw" or "Toolbox"
            </p>
            <Link to="/definitions" className={styles.stepCta}>
              <PlusIcon width={14} height={14} />
              Create first definition
            </Link>
          </div>
        </div>
        <div className={styles.onboardingStep}>
          <span className={styles.stepNumber}>3</span>
          <div className={styles.stepContent}>
            <p className={styles.stepTitle}>Stock your items</p>
            <p className={styles.stepDesc}>
              Put quantities into locations — the actual physical inventory
            </p>
            <Link to="/locations" className={styles.stepCta}>
              <PlusIcon width={14} height={14} />
              Add first item
            </Link>
          </div>
        </div>
      </div>
    </div>
  )
}

function DashboardSkeleton() {
  return (
    <>
      <div className={styles.skeletonSearch} />
      <div className={styles.statGrid}>
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className={styles.skeletonCard} />
        ))}
      </div>
      <div className={styles.skeletonSection}>
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className={styles.skeletonRow} />
        ))}
      </div>
    </>
  )
}
