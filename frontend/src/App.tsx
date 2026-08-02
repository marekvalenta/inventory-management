import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { ToastProvider } from './context/ToastContext'
import { OnlineStatusProvider } from './context/OnlineStatusContext'
import { OfflineBanner } from './components/common/OfflineBanner'
import { ToastContainer } from './components/common/Toast'
import { AppLayout } from './components/layout/AppLayout'
import { LocationsPage } from './pages/LocationsPage'
import { LocationDetailPage } from './pages/LocationDetailPage'
import { DefinitionListPage } from './pages/DefinitionListPage'
import { DefinitionDetailPage } from './pages/DefinitionDetailPage'
import { TagsPage } from './pages/TagsPage'
import { HomePage } from './pages/HomePage'
import { NotFoundPage } from './pages/NotFoundPage'

const queryClient = new QueryClient()

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <OnlineStatusProvider>
        <ToastProvider>
          <BrowserRouter>
            <OfflineBanner />
            <ToastContainer />
            <Routes>
              <Route element={<AppLayout />}>
                <Route index element={<Navigate to="/locations" replace />} />
                <Route path="locations" element={<LocationsPage />} />
                <Route path="locations/:id" element={<LocationDetailPage />} />
                <Route path="definitions" element={<DefinitionListPage />} />
                <Route path="definitions/:id" element={<DefinitionDetailPage />} />
                <Route path="instances/:id" element={<HomePage />} />
                <Route path="tags" element={<TagsPage />} />
                <Route path="settings" element={<HomePage />} />
                <Route path="*" element={<NotFoundPage />} />
              </Route>
            </Routes>
          </BrowserRouter>
        </ToastProvider>
      </OnlineStatusProvider>
    </QueryClientProvider>
  )
}

export default App
