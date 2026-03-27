import { Outlet, Link, useLocation, useNavigate } from 'react-router-dom'
import { Book, FileText, Settings, LogOut, MessageSquare, PanelLeftClose, PanelLeftOpen, Brain } from 'lucide-react'
import { useAuth } from '@/contexts/AuthContext'
import { Button } from '@/components/ui/button'
import { useState } from 'react'

export default function Layout() {
  const { logout } = useAuth()
  const location = useLocation()
  const navigate = useNavigate()
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)

  const navItems = [
    { path: '/kbs', icon: Book, label: '知识库' },
    { path: '/global-chat', icon: Brain, label: '全局聊天' },
    { path: '/files', icon: FileText, label: '文件' },
    { path: '/settings', icon: Settings, label: '设置' },
  ]

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  return (
    <div className="flex h-screen overflow-hidden bg-gradient-to-b from-background via-background to-secondary/20">
      {/* Sidebar - Desktop */}
      <aside className={`hidden min-h-screen border-r border-border/50 glass md:block transition-all duration-300 ${
        sidebarCollapsed ? 'w-20' : 'w-64'
      }`}>
        <div className="flex flex-col h-full">
          {/* Logo Area */}
          <div className={`flex items-center gap-3 p-4 border-b border-border/50 ${sidebarCollapsed ? 'justify-center' : ''}`}>
            <div className="relative">
              <div className="absolute inset-0 bg-primary/20 blur-lg rounded-full" />
              <Book className="h-7 w-7 text-primary relative z-10" />
            </div>
            {!sidebarCollapsed && (
              <div>
                <span className="text-xl font-semibold bg-gradient-to-r from-primary to-accent bg-clip-text text-transparent">
                  myRAG
                </span>
                <p className="text-xs text-muted-foreground -mt-1">智能知识库</p>
              </div>
            )}
          </div>

          {/* Collapse Toggle Button */}
          <div className="flex justify-end p-2 border-b border-border/50">
            <Button
              variant="ghost"
              size="icon"
              onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
              className="h-8 w-8 hover:bg-secondary/50"
            >
              {sidebarCollapsed ? <PanelLeftOpen className="h-4 w-4" /> : <PanelLeftClose className="h-4 w-4" />}
            </Button>
          </div>

          {/* Navigation */}
          <nav className="flex-1 flex flex-col gap-1.5 p-4 overflow-y-auto">
            {navItems.map((item) => {
              const Icon = item.icon
              const isActive = location.pathname.startsWith(item.path)
              return (
                <Link
                  key={item.path}
                  to={item.path}
                  className={`group flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-all ${
                    isActive
                      ? 'bg-gradient-to-r from-primary/10 to-primary/5 text-primary shadow-sm ring-1 ring-primary/20'
                      : 'text-muted-foreground hover:bg-secondary/50 hover:text-foreground'
                  } ${sidebarCollapsed ? 'justify-center' : ''}`}
                  title={sidebarCollapsed ? item.label : undefined}
                >
                  <Icon className={`h-5 w-5 transition-transform group-hover:scale-105 ${isActive ? 'text-primary' : ''}`} />
                  {!sidebarCollapsed && item.label}
                </Link>
              )
            })}
          </nav>

          {/* Sidebar Footer */}
          {!sidebarCollapsed && (
            <div className="p-4 border-t border-border/50">
              <div className="rounded-xl bg-gradient-to-br from-primary/5 to-accent/5 p-4 border border-border/50">
                <div className="flex items-center gap-2 mb-2">
                  <MessageSquare className="h-4 w-4 text-primary" />
                  <span className="text-xs font-semibold text-foreground">AI 助手</span>
                </div>
                <p className="text-xs text-muted-foreground">
                  基于 RAG 技术的智能问答
                </p>
              </div>
            </div>
          )}

          {/* Logout Button */}
          <div className="p-4 border-t border-border/50">
            <Button
              variant="ghost"
              onClick={handleLogout}
              className={`w-full justify-center ${sidebarCollapsed ? '' : 'justify-start gap-3'}`}
            >
              <LogOut className="h-5 w-5" />
              {!sidebarCollapsed && <span>退出登录</span>}
            </Button>
          </div>
        </div>
      </aside>

      {/* Sidebar - Mobile */}
      {mobileMenuOpen && (
        <aside className="fixed inset-y-0 left-0 z-40 w-64 glass md:hidden">
          <div className="flex flex-col h-full">
            <div className="flex items-center gap-3 p-4 pt-6 border-b border-border/50">
              <div className="relative">
                <div className="absolute inset-0 bg-primary/20 blur-lg rounded-full" />
                <Book className="h-7 w-7 text-primary relative z-10" />
              </div>
              <div>
                <span className="text-xl font-semibold bg-gradient-to-r from-primary to-accent bg-clip-text text-transparent">
                  myRAG
                </span>
                <p className="text-xs text-muted-foreground -mt-1">智能知识库</p>
              </div>
            </div>
            <nav className="flex-1 flex flex-col gap-1.5 p-4 overflow-y-auto">
              {navItems.map((item) => {
                const Icon = item.icon
                const isActive = location.pathname.startsWith(item.path)
                return (
                  <Link
                    key={item.path}
                    to={item.path}
                    onClick={() => setMobileMenuOpen(false)}
                    className={`flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-all ${
                      isActive
                        ? 'bg-gradient-to-r from-primary/10 to-primary/5 text-primary shadow-sm ring-1 ring-primary/20'
                        : 'text-muted-foreground hover:bg-secondary/50 hover:text-foreground'
                    }`}
                  >
                    <Icon className={`h-5 w-5 ${isActive ? 'text-primary' : ''}`} />
                    {item.label}
                  </Link>
                )
              })}
            </nav>
            <div className="p-4 border-t border-border/50">
              <Button
                variant="ghost"
                onClick={handleLogout}
                className="w-full justify-start gap-3"
              >
                <LogOut className="h-5 w-5" />
                <span>退出登录</span>
              </Button>
            </div>
          </div>
        </aside>
      )}

      {/* Main Content */}
      <main className="flex-1 min-w-0 overflow-hidden">
        <div className="h-full overflow-y-auto p-4 lg:p-6">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
