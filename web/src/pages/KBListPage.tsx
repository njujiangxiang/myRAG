import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { Plus, MessageSquare, MoreVertical, Trash2, Book, FileText, Check } from 'lucide-react'
import { useAuth } from '@/contexts/AuthContext'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

interface KnowledgeBase {
  id: string
  name: string
  description: string
  document_count?: number
  rag_type?: string
  created_at: string
}

export default function KBListPage() {
  const [kbs, setKbs] = useState<KnowledgeBase[]>([])
  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState(false)
  const [newName, setNewName] = useState('')
  const [newDesc, setNewDesc] = useState('')
  const [newRagType, setNewRagType] = useState('vector')
  const [showCreate, setShowCreate] = useState(false)

  const { token } = useAuth()

  const fetchKBs = async () => {
    try {
      const response = await fetch('/api/v1/kbs', {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      })
      if (response.ok) {
        const data = await response.json()
        setKbs(data)
      }
    } catch (error) {
      console.error('Failed to fetch KBs:', error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchKBs()
  }, [token])

  const createKB = async () => {
    if (!newName.trim()) return

    setCreating(true)
    try {
      const response = await fetch('/api/v1/kbs', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          name: newName,
          description: newDesc,
          rag_type: newRagType,
        }),
      })

      if (response.ok) {
        setShowCreate(false)
        setNewName('')
        setNewDesc('')
        setNewRagType('vector')
        fetchKBs()
      }
    } catch (error) {
      console.error('Failed to create KB:', error)
    } finally {
      setCreating(false)
    }
  }

  const deleteKB = async (id: string) => {
    if (!confirm('确定要删除这个知识库吗？')) return

    try {
      const response = await fetch(`/api/v1/kbs/${id}`, {
        method: 'DELETE',
        headers: {
          Authorization: `Bearer ${token}`,
        },
      })

      if (response.ok) {
        fetchKBs()
      }
    } catch (error) {
      console.error('Failed to delete KB:', error)
    }
  }

  return (
    <div className="p-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold bg-gradient-to-r from-primary to-accent bg-clip-text text-transparent">
            知识库
          </h1>
          <p className="text-muted-foreground mt-1">管理您的知识库和文档</p>
        </div>
        <Button
          onClick={() => setShowCreate(!showCreate)}
          className="bg-gradient-to-r from-primary to-accent hover:from-primary/90 hover:to-accent/90 shadow-lg shadow-primary/25 rounded-xl"
        >
          <Plus className="h-4 w-4 mr-2" />
          创建知识库
        </Button>
      </div>

      {/* Create Form */}
      {showCreate && (
        <Card className="mb-8 glass border-border/50 shadow-xl">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <div className="w-8 h-8 rounded-xl bg-gradient-to-br from-primary/90 to-accent/90 flex items-center justify-center">
                <Plus className="h-4 w-4 text-white" />
              </div>
              创建新知识库
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium text-foreground ml-1">
                名称
              </label>
              <Input
                id="name"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                placeholder="输入知识库名称"
                className="bg-background/50 border-border/50 focus:border-primary/50 focus:ring-primary/20 rounded-xl h-11"
              />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium text-foreground ml-1">
                描述
              </label>
              <Textarea
                id="description"
                value={newDesc}
                onChange={(e) => setNewDesc(e.target.value)}
                placeholder="输入知识库描述（可选）"
                className="bg-background/50 border-border/50 focus:border-primary/50 focus:ring-primary/20 rounded-xl min-h-[100px]"
              />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium text-foreground ml-1">
                RAG 类型
              </label>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="outline"
                    className="w-full justify-between bg-background/50 border-border/50 hover:bg-background/80 rounded-xl h-11"
                  >
                    {newRagType === 'vector' && '向量检索（默认）'}
                    {newRagType === 'graph' && '知识图谱（实验性）'}
                    {newRagType === 'hybrid' && '混合检索（向量 + 关键词）'}
                    {newRagType === 'keyword' && '关键词检索（BM25）'}
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent className="w-full">
                  <DropdownMenuItem onClick={() => setNewRagType('vector')}>
                    <div className="flex items-center justify-between w-full">
                      <span>向量检索（默认）</span>
                      {newRagType === 'vector' && <Check className="h-4 w-4" />}
                    </div>
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => setNewRagType('graph')}>
                    <div className="flex items-center justify-between w-full">
                      <span>知识图谱（实验性）</span>
                      {newRagType === 'graph' && <Check className="h-4 w-4" />}
                    </div>
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => setNewRagType('hybrid')}>
                    <div className="flex items-center justify-between w-full">
                      <span>混合检索（向量 + 关键词）</span>
                      {newRagType === 'hybrid' && <Check className="h-4 w-4" />}
                    </div>
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => setNewRagType('keyword')}>
                    <div className="flex items-center justify-between w-full">
                      <span>关键词检索（BM25）</span>
                      {newRagType === 'keyword' && <Check className="h-4 w-4" />}
                    </div>
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
              <p className="text-xs text-muted-foreground ml-1">
                {newRagType === 'vector' && '使用向量相似度检索，适合语义搜索场景'}
                {newRagType === 'graph' && '基于知识图谱的检索增强，v1.1 功能，当前使用向量检索替代'}
                {newRagType === 'hybrid' && '结合向量相似度和关键词匹配，综合两种方法的优势'}
                {newRagType === 'keyword' && '使用 BM25 算法进行关键词检索，适合精确匹配场景'}
              </p>
            </div>
          </CardContent>
          <CardFooter className="gap-2">
            <Button
              onClick={createKB}
              disabled={creating || !newName.trim()}
              className="bg-gradient-to-r from-primary to-accent hover:from-primary/90 hover:to-accent/90 rounded-xl"
            >
              {creating ? (
                <span className="flex items-center gap-2">
                  <span className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                  创建中...
                </span>
              ) : (
                '创建知识库'
              )}
            </Button>
            <Button
              variant="outline"
              onClick={() => setShowCreate(false)}
              className="rounded-xl"
            >
              取消
            </Button>
          </CardFooter>
        </Card>
      )}

      {/* Empty State */}
      {loading ? (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map((i) => (
            <Card key={i} className="animate-pulse glass border-border/50">
              <CardHeader>
                <div className="h-4 bg-muted rounded w-3/4" />
                <div className="h-3 bg-muted rounded w-1/2" />
              </CardHeader>
              <CardContent>
                <div className="h-3 bg-muted rounded w-full" />
              </CardContent>
            </Card>
          ))}
        </div>
      ) : kbs.length === 0 ? (
        <Card className="glass border-border/50 shadow-xl">
          <CardContent className="flex flex-col items-center justify-center py-16">
            <div className="w-16 h-16 rounded-2xl bg-gradient-to-br from-primary/10 to-accent/10 flex items-center justify-center mb-6">
              <Book className="h-8 w-8 text-primary" />
            </div>
            <h3 className="text-xl font-semibold mb-2">暂无知识库</h3>
            <p className="text-muted-foreground text-center mb-6 max-w-md">
              创建您的第一个知识库，开始上传文档并与您的数据对话。基于 RAG 技术的智能问答系统让知识触手可及。
            </p>
            <Button
              onClick={() => setShowCreate(true)}
              className="bg-gradient-to-r from-primary to-accent hover:from-primary/90 hover:to-accent/90 shadow-lg shadow-primary/25 rounded-xl"
            >
              <Plus className="h-4 w-4 mr-2" />
              创建知识库
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {kbs.map((kb) => (
            <Card key={kb.id} className="flex flex-col glass border-border/50 shadow-lg hover:shadow-xl transition-all duration-300 group">
              <CardHeader>
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <CardTitle className="line-clamp-1 group-hover:text-primary transition-colors">
                      {kb.name}
                    </CardTitle>
                    <CardDescription className="line-clamp-2 mt-1">
                      {kb.description || '暂无描述'}
                    </CardDescription>
                    <div className="mt-2">
                      <span className={`inline-flex px-2 py-1 text-xs rounded-full ${
                        kb.rag_type === 'vector' ? 'bg-blue-500/10 text-blue-500' :
                        kb.rag_type === 'graph' ? 'bg-purple-500/10 text-purple-500' :
                        kb.rag_type === 'hybrid' ? 'bg-green-500/10 text-green-500' :
                        kb.rag_type === 'keyword' ? 'bg-orange-500/10 text-orange-500' :
                        'bg-gray-500/10 text-gray-500'
                      }`}>
                        {kb.rag_type === 'vector' && '向量检索'}
                        {kb.rag_type === 'graph' && '知识图谱'}
                        {kb.rag_type === 'hybrid' && '混合检索'}
                        {kb.rag_type === 'keyword' && '关键词检索'}
                        {!kb.rag_type && '向量检索'}
                      </span>
                    </div>
                  </div>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button variant="ghost" size="icon" className="h-8 w-8 rounded-xl opacity-0 group-hover:opacity-100 transition-opacity">
                        <MoreVertical className="h-4 w-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem
                        onClick={() => deleteKB(kb.id)}
                        className="text-destructive"
                      >
                        <Trash2 className="h-4 w-4 mr-2" />
                        删除
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              </CardHeader>
              <CardContent className="flex-1">
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <FileText className="h-4 w-4" />
                  {kb.document_count || 0} 个文档
                </div>
              </CardContent>
              <CardFooter className="gap-2 pt-4 border-t border-border/50">
                <Link to={`/kbs/${kb.id}/chat`} className="flex-1">
                  <Button
                    variant="default"
                    className="w-full rounded-xl bg-gradient-to-r from-primary to-accent hover:from-primary/90 hover:to-accent/90 shadow-md"
                  >
                    <MessageSquare className="h-4 w-4 mr-2" />
                    聊天
                  </Button>
                </Link>
                <Link to={`/kbs/${kb.id}`} className="flex-1">
                  <Button
                    variant="outline"
                    className="w-full rounded-xl"
                  >
                    管理文档
                  </Button>
                </Link>
              </CardFooter>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
