import { useState } from 'react'
import { User, LogOut, Key, Bell, Shield } from 'lucide-react'
import { useAuth } from '@/contexts/AuthContext'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'

export default function SettingsPage() {
  const { user, logout } = useAuth()
  const [loading, setLoading] = useState(false)
  const [notifications, setNotifications] = useState({
    email: true,
    desktop: true,
    documentReady: true,
    processingComplete: true,
  })

  const handleSaveProfile = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    // TODO: Implement profile update
    setTimeout(() => setLoading(false), 1000)
  }

  const handleSaveNotifications = async () => {
    setLoading(true)
    // TODO: Implement notification settings update
    setTimeout(() => setLoading(false), 1000)
  }

  return (
    <div className="p-6 max-w-4xl mx-auto">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold bg-gradient-to-r from-primary to-accent bg-clip-text text-transparent">
          设置
        </h1>
        <p className="text-muted-foreground mt-1">管理您的账户和偏好设置</p>
      </div>

      {/* Profile Section */}
      <Card className="mb-6 glass border-border/50 shadow-xl">
        <CardHeader>
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-primary/10 to-accent/10 flex items-center justify-center">
              <User className="h-5 w-5 text-primary" />
            </div>
            <div>
              <CardTitle>个人资料</CardTitle>
              <CardDescription>更新您的账户信息</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSaveProfile} className="space-y-4">
            <div className="grid gap-2">
              <Label htmlFor="email">邮箱地址</Label>
              <Input
                id="email"
                type="email"
                value={user?.email || ''}
                disabled
                className="bg-muted/50 rounded-xl"
              />
              <p className="text-xs text-muted-foreground">邮箱地址不能修改</p>
            </div>
            <div className="grid gap-2">
              <Label htmlFor="userId">用户 ID</Label>
              <Input
                id="userId"
                value={user?.id || ''}
                disabled
                className="bg-muted/50 rounded-xl font-mono text-xs"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="role">角色</Label>
              <Input
                id="role"
                value={user?.role === 'user' ? '普通用户' : user?.role === 'admin' ? '管理员' : user?.role}
                disabled
                className="bg-muted/50 rounded-xl"
              />
            </div>
            <div className="flex items-center gap-2 pt-4">
              <Button type="submit" disabled={loading} className="rounded-xl">
                {loading ? '保存中...' : '保存更改'}
              </Button>
              <Button type="button" variant="outline" className="rounded-xl">
                修改密码
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      {/* Notification Settings */}
      <Card className="mb-6 glass border-border/50 shadow-xl">
        <CardHeader>
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-primary/10 to-accent/10 flex items-center justify-center">
              <Bell className="h-5 w-5 text-primary" />
            </div>
            <div>
              <CardTitle>通知设置</CardTitle>
              <CardDescription>管理您接收的通知类型</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label>邮件通知</Label>
              <p className="text-sm text-muted-foreground">接收系统邮件通知</p>
            </div>
            <Switch
              checked={notifications.email}
              onCheckedChange={(checked: boolean) => setNotifications({...notifications, email: checked})}
            />
          </div>
          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label>桌面通知</Label>
              <p className="text-sm text-muted-foreground">显示桌面弹窗通知</p>
            </div>
            <Switch
              checked={notifications.desktop}
              onCheckedChange={(checked: boolean) => setNotifications({...notifications, desktop: checked})}
            />
          </div>
          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label>文档就绪通知</Label>
              <p className="text-sm text-muted-foreground">文档处理完成后通知</p>
            </div>
            <Switch
              checked={notifications.documentReady}
              onCheckedChange={(checked: boolean) => setNotifications({...notifications, documentReady: checked})}
            />
          </div>
          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label>处理完成通知</Label>
              <p className="text-sm text-muted-foreground">知识库处理完成后通知</p>
            </div>
            <Switch
              checked={notifications.processingComplete}
              onCheckedChange={(checked: boolean) => setNotifications({...notifications, processingComplete: checked})}
            />
          </div>
          <Button onClick={handleSaveNotifications} disabled={loading} className="w-full rounded-xl">
            保存通知设置
          </Button>
        </CardContent>
      </Card>

      {/* API Keys Section */}
      <Card className="mb-6 glass border-border/50 shadow-xl">
        <CardHeader>
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-primary/10 to-accent/10 flex items-center justify-center">
              <Key className="h-5 w-5 text-primary" />
            </div>
            <div>
              <CardTitle>API 密钥</CardTitle>
              <CardDescription>管理您的 API 访问密钥</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <div className="flex items-center justify-between p-4 rounded-xl bg-secondary/50 border border-border/50">
              <div className="flex items-center gap-3">
                <div className="w-2 h-2 bg-green-500 rounded-full" />
                <div>
                  <p className="font-medium">默认密钥</p>
                  <p className="text-sm text-muted-foreground font-mono">sk-••••••••••••</p>
                </div>
              </div>
              <Button variant="outline" size="sm" className="rounded-xl">
                管理
              </Button>
            </div>
            <Button variant="outline" className="w-full rounded-xl">
              创建新密钥
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Danger Zone */}
      <Card className="border-destructive/50 shadow-xl">
        <CardHeader>
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-destructive/10 flex items-center justify-center">
              <Shield className="h-5 w-5 text-destructive" />
            </div>
            <div>
              <CardTitle className="text-destructive">危险区域</CardTitle>
              <CardDescription>这些操作不可撤销，请谨慎操作</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between p-4 rounded-xl bg-destructive/5 border border-destructive/20">
            <div>
              <p className="font-medium">退出登录</p>
              <p className="text-sm text-muted-foreground">退出当前账户</p>
            </div>
            <Button variant="destructive" onClick={logout} className="rounded-xl">
              <LogOut className="h-4 w-4 mr-2" />
              退出
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
