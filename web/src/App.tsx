import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { Toaster } from "@/components/ui/toaster"
import AuthProvider, { useAuth } from '@/contexts/AuthContext'
import Layout from '@/components/Layout'
import LoginPage from '@/pages/LoginPage'
import KBListPage from '@/pages/KBListPage'
import KBDetailPage from '@/pages/KBDetailPage'
import ChatPage from '@/pages/ChatPage'
import GlobalChatPage from '@/pages/GlobalChatPage'
import FilesPage from '@/pages/FilesPage'
import SettingsPage from '@/pages/SettingsPage'

function AppRoutes() {
  const { isAuthenticated, isLoading } = useAuth()

  if (isLoading) {
    return (
      <div style={{
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        height: '100vh',
        width: '100vw'
      }}>
        <div style={{
          fontSize: '16px',
          color: '#666'
        }}>
          加载中...
        </div>
      </div>
    )
  }

  return (
    <Routes>
      {!isAuthenticated ? (
        <>
          <Route path="/login" element={<LoginPage />} />
          <Route path="*" element={<Navigate to="/login" replace />} />
        </>
      ) : (
        <>
          <Route path="/" element={<Layout />}>
            <Route index element={<Navigate to="/kbs" replace />} />
            <Route path="kbs" element={<KBListPage />} />
            <Route path="kbs/:id" element={<KBDetailPage />} />
            <Route path="kbs/:id/chat" element={<ChatPage />} />
            <Route path="global-chat" element={<GlobalChatPage />} />
            <Route path="files" element={<FilesPage />} />
            <Route path="settings" element={<SettingsPage />} />
          </Route>
        </>
      )}
    </Routes>
  )
}

function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <AppRoutes />
        <Toaster />
      </BrowserRouter>
    </AuthProvider>
  )
}

export default App
