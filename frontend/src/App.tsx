import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { ToastProvider } from './context/ToastContext'
import { OnlineStatusProvider } from './context/OnlineStatusContext'
import { OfflineBanner } from './components/common/OfflineBanner'
import { ToastContainer } from './components/common/Toast'
import { AppLayout } from './components/layout/AppLayout'
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
                <Route path="locations" element={<HomePage />} />
                <Route path="locations/:id" element={<HomePage />} />
                <Route path="definitions" element={<HomePage />} />
                <Route path="definitions/:id" element={<HomePage />} />
                <Route path="instances/:id" element={<HomePage />} />
                <Route path="tags" element={<HomePage />} />
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
