import { useState, useEffect } from 'react'
import { FileText, Search, Trash2, Download, Eye } from 'lucide-react'
import { useAuth } from '@/contexts/AuthContext'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'

interface Document {
  id: string
  kb_id: string
  kb_name: string
  filename: string
  file_size: number
  status: 'pending' | 'processing' | 'indexed' | 'error'
  created_at: string
}

const statusConfig = {
  pending: { label: '等待中', color: 'text-yellow-500 bg-yellow-500/10' },
  processing: { label: '处理中', color: 'text-blue-500 bg-blue-500/10' },
  indexed: { label: '已索引', color: 'text-green-500 bg-green-500/10' },
  error: { label: '错误', color: 'text-red-500 bg-red-500/10' },
}

export default function FilesPage() {
  const [documents, setDocuments] = useState<Document[]>([])
  const [loading, setLoading] = useState(true)
  const [searchQuery, setSearchQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('all')
  const { token } = useAuth()

  const fetchAllDocuments = async () => {
    try {
      // First, get all knowledge bases
      const kbsResponse = await fetch('/api/v1/kbs', {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      })

      if (!kbsResponse.ok) return

      const kbs = await kbsResponse.json()

      // Then, get documents for each KB
      const allDocs: Document[] = []
      for (const kb of kbs) {
        const docsResponse = await fetch(`/api/v1/kbs/${kb.id}/docs`, {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        })

        if (docsResponse.ok) {
          const docs = await docsResponse.json()
          const docsWithKbName = docs.map((doc: any) => ({
            ...doc,
            kb_id: kb.id,
            kb_name: kb.name,
          }))
          allDocs.push(...docsWithKbName)
        }
      }

      setDocuments(allDocs)
    } catch (error) {
      console.error('Failed to fetch documents:', error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchAllDocuments()
    // Poll for status updates
    const interval = setInterval(fetchAllDocuments, 10000)
    return () => clearInterval(interval)
  }, [token])

  const deleteDocument = async (docId: string) => {
    if (!confirm('确定要删除这个文档吗？')) return

    try {
      const response = await fetch(`/api/v1/docs/${docId}`, {
        method: 'DELETE',
        headers: {
          Authorization: `Bearer ${token}`,
        },
      })

      if (response.ok) {
        fetchAllDocuments()
      }
    } catch (error) {
      console.error('Failed to delete:', error)
    }
  }

  const formatFileSize = (bytes: number) => {
    if (bytes < 1024) return bytes + ' B'
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
  }

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString('zh-CN')
  }

  const filteredDocuments = documents.filter(doc => {
    const matchesSearch = doc.filename.toLowerCase().includes(searchQuery.toLowerCase()) ||
                         doc.kb_name.toLowerCase().includes(searchQuery.toLowerCase())
    const matchesStatus = statusFilter === 'all' || doc.status === statusFilter
    return matchesSearch && matchesStatus
  })

  return (
    <div className="p-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold bg-gradient-to-r from-primary to-accent bg-clip-text text-transparent">
            文件管理
          </h1>
          <p className="text-muted-foreground mt-1">查看所有知识库中的文档</p>
        </div>
        <div className="flex items-center gap-3">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="搜索文件..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-10 w-64 rounded-xl bg-background/50 border-border/50"
            />
          </div>
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="h-11 px-4 rounded-xl border border-border/50 bg-background/50 text-sm focus:outline-none focus:ring-2 focus:ring-primary/20"
          >
            <option value="all">全部状态</option>
            <option value="pending">等待中</option>
            <option value="processing">处理中</option>
            <option value="indexed">已索引</option>
            <option value="error">错误</option>
          </select>
        </div>
      </div>

      {/* Stats Cards */}
      <div className="grid gap-4 md:grid-cols-4 mb-8">
        <Card className="glass border-border/50">
          <CardContent className="pt-6">
            <div className="flex items-center gap-4">
              <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-primary/10 to-accent/10 flex items-center justify-center">
                <FileText className="h-6 w-6 text-primary" />
              </div>
              <div>
                <p className="text-2xl font-bold">{documents.length}</p>
                <p className="text-sm text-muted-foreground">总文档数</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card className="glass border-border/50">
          <CardContent className="pt-6">
            <div className="flex items-center gap-4">
              <div className="w-12 h-12 rounded-xl bg-green-500/10 flex items-center justify-center">
                <FileText className="h-6 w-6 text-green-500" />
              </div>
              <div>
                <p className="text-2xl font-bold">{documents.filter(d => d.status === 'indexed').length}</p>
                <p className="text-sm text-muted-foreground">已索引</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card className="glass border-border/50">
          <CardContent className="pt-6">
            <div className="flex items-center gap-4">
              <div className="w-12 h-12 rounded-xl bg-blue-500/10 flex items-center justify-center">
                <FileText className="h-6 w-6 text-blue-500" />
              </div>
              <div>
                <p className="text-2xl font-bold">{documents.filter(d => d.status === 'processing').length}</p>
                <p className="text-sm text-muted-foreground">处理中</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card className="glass border-border/50">
          <CardContent className="pt-6">
            <div className="flex items-center gap-4">
              <div className="w-12 h-12 rounded-xl bg-red-500/10 flex items-center justify-center">
                <FileText className="h-6 w-6 text-red-500" />
              </div>
              <div>
                <p className="text-2xl font-bold">{documents.filter(d => d.status === 'error').length}</p>
                <p className="text-sm text-muted-foreground">错误</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Documents List */}
      <Card className="glass border-border/50 shadow-xl">
        <CardContent className="p-0">
          {loading ? (
            <div className="space-y-2 p-4">
              {[1, 2, 3, 4, 5].map((i) => (
                <div key={i} className="flex items-center gap-4 p-4 rounded-xl border animate-pulse">
                  <div className="w-10 h-10 rounded-xl bg-muted" />
                  <div className="flex-1 space-y-2">
                    <div className="h-4 bg-muted rounded w-3/4" />
                    <div className="h-3 bg-muted rounded w-1/2" />
                  </div>
                </div>
              ))}
            </div>
          ) : filteredDocuments.length === 0 ? (
            <div className="text-center py-16">
              <div className="w-16 h-16 rounded-2xl bg-muted flex items-center justify-center mx-auto mb-4">
                <FileText className="h-8 w-8 text-muted-foreground opacity-50" />
              </div>
              <p className="font-semibold text-lg mb-2">暂无文档</p>
              <p className="text-sm text-muted-foreground">上传文档后会显示在这里</p>
            </div>
          ) : (
            <div className="divide-y divide-border/50">
              {filteredDocuments.map((doc) => {
                const statusColors = statusConfig[doc.status]
                return (
                  <div
                    key={doc.id}
                    className="flex items-center gap-4 p-4 hover:bg-secondary/50 transition-all group"
                  >
                    <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-primary/10 to-accent/10 flex items-center justify-center flex-shrink-0">
                      <FileText className="h-5 w-5 text-primary" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <p className="font-medium truncate">{doc.filename}</p>
                        <span className="text-xs text-muted-foreground px-2 py-0.5 rounded-full bg-secondary/50">
                          {doc.kb_name}
                        </span>
                      </div>
                      <p className="text-sm text-muted-foreground">
                        {formatFileSize(doc.file_size)} · {formatDate(doc.created_at)}
                      </p>
                    </div>
                    <div className={`flex items-center gap-2 px-3 py-1.5 rounded-full ${statusColors.color}`}>
                      <span className="text-sm font-medium">{statusConfig[doc.status].label}</span>
                    </div>
                    <div className="flex items-center gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                      <Button variant="ghost" size="icon" className="rounded-xl">
                        <Eye className="h-4 w-4" />
                      </Button>
                      <Button variant="ghost" size="icon" className="rounded-xl">
                        <Download className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => deleteDocument(doc.id)}
                        className="rounded-xl hover:bg-destructive/10 hover:text-destructive"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
