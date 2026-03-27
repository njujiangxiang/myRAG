import { useState, useEffect, useRef } from 'react'
import { Send, MessageSquare, FileText, Brain } from 'lucide-react'
import { useAuth } from '@/contexts/AuthContext'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'

interface Message {
  id: string
  role: 'user' | 'assistant'
  content: string
  metadata?: {
    sources?: string[]
    kb_names?: string[]
    context_chunks?: number
  }
}

interface KnowledgeBase {
  id: string
  name: string
  description: string
  document_count?: number
  created_at: string
}

export default function GlobalChatPage() {
  const { token } = useAuth()
  const [kbs, setKbs] = useState<KnowledgeBase[]>([])
  const [selectedKBs, setSelectedKBs] = useState<Set<string>>(new Set())
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [loadingKBs, setLoadingKBs] = useState(true)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  useEffect(() => {
    scrollToBottom()
  }, [messages])

  useEffect(() => {
    fetchKBs()
  }, [token])

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
        // 默认选中所有知识库
        const allIds = new Set<string>(data.map((kb: KnowledgeBase) => kb.id))
        setSelectedKBs(allIds)
      }
    } catch (error) {
      console.error('Failed to fetch KBs:', error)
    } finally {
      setLoadingKBs(false)
    }
  }

  const toggleKB = (id: string) => {
    setSelectedKBs((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }

  const sendMessage = async () => {
    if (!input.trim() || selectedKBs.size === 0) return

    const userMessage: Message = {
      id: Date.now().toString(),
      role: 'user',
      content: input,
    }

    setMessages((prev) => [...prev, userMessage])
    setInput('')
    setLoading(true)

    try {
      const response = await fetch('/api/v1/chat', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          content: input,
          kb_ids: Array.from(selectedKBs),
        }),
      })

      if (!response.ok) {
        throw new Error('请求失败')
      }

      const data = await response.json()
      const assistantMessage: Message = {
        id: data.message_id,
        role: 'assistant',
        content: data.content,
        metadata: data.metadata,
      }

      setMessages((prev) => [...prev, assistantMessage])
    } catch (error) {
      console.error('Failed to send message:', error)
      setMessages((prev) => [
        ...prev,
        {
          id: 'error',
          role: 'assistant',
          content: '抱歉，发生错误。请确保已选择至少一个知识库并稍后重试。',
        },
      ])
    } finally {
      setLoading(false)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      sendMessage()
    }
  }

  const totalDocs = kbs
    .filter((kb) => selectedKBs.has(kb.id))
    .reduce((sum, kb) => sum + (kb.document_count || 0), 0)

  return (
    <div className="flex flex-col h-[calc(100vh-4rem)] overflow-hidden">
      {/* Header */}
      <div className="flex-shrink-0 p-6 border-b border-border/50 glass">
        <div className="flex items-center gap-3 mb-2">
          <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-primary/90 to-accent/90 flex items-center justify-center">
            <Brain className="h-5 w-5 text-white" />
          </div>
          <div>
            <h1 className="text-3xl font-bold bg-gradient-to-r from-primary to-accent bg-clip-text text-transparent">
              全局聊天
            </h1>
            <p className="text-muted-foreground">跨多个知识库进行智能检索和对话</p>
          </div>
        </div>
      </div>

      <div className="flex flex-1 overflow-hidden">
        {/* KB Selection - Sidebar */}
        <div className="w-80 flex-shrink-0 border-r border-border/50 overflow-y-auto p-4 bg-background/50">
          <Card className="glass border-border/50 shadow-xl">
            <CardHeader>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="w-8 h-8 rounded-xl bg-gradient-to-br from-primary/10 to-accent/10 flex items-center justify-center">
                    <FileText className="h-4 w-4 text-primary" />
                  </div>
                  <div>
                    <CardTitle className="text-sm">选择知识库</CardTitle>
                    <CardDescription className="text-xs">
                      {selectedKBs.size} 个 ({totalDocs} 文档)
                    </CardDescription>
                  </div>
                </div>
              </div>
            </CardHeader>
            <CardContent className="p-3 pt-0">
              {loadingKBs ? (
                <div className="space-y-3">
                  {[1, 2, 3].map((i) => (
                    <div key={i} className="flex items-center gap-4 p-3 rounded-xl border animate-pulse">
                      <div className="w-5 h-5 bg-muted rounded" />
                      <div className="flex-1">
                        <div className="h-4 bg-muted rounded w-3/4 mb-2" />
                        <div className="h-3 bg-muted rounded w-1/2" />
                      </div>
                    </div>
                  ))}
                </div>
              ) : kbs.length === 0 ? (
                <div className="text-center py-8">
                  <p className="text-muted-foreground text-sm">暂无知识库，请先创建知识库</p>
                </div>
              ) : (
                <div className="space-y-2 max-h-[400px] overflow-y-auto">
                  {kbs.map((kb) => (
                    <div
                      key={kb.id}
                      onClick={() => toggleKB(kb.id)}
                      className={`flex items-start gap-3 p-3 rounded-xl border cursor-pointer transition-all ${
                        selectedKBs.has(kb.id)
                          ? 'bg-primary/5 border-primary/50 shadow-md'
                          : 'bg-secondary/30 border-border/50 hover:border-primary/30'
                      }`}
                    >
                      <Checkbox
                        checked={selectedKBs.has(kb.id)}
                        onCheckedChange={() => toggleKB(kb.id)}
                        className="mt-1"
                      />
                      <div className="flex-1 min-w-0">
                        <p className="font-medium text-sm truncate">{kb.name}</p>
                        <p className="text-xs text-muted-foreground truncate">
                          {kb.description || '暂无描述'}
                        </p>
                        <p className="text-xs text-muted-foreground mt-1">
                          {kb.document_count || 0} 个文档
                        </p>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Chat Area */}
        <div className="flex-1 flex flex-col overflow-hidden">
          {/* Messages */}
          <div className="flex-1 overflow-y-auto p-4 space-y-4">
            {messages.length === 0 ? (
              <div className="flex flex-col items-center justify-center h-full py-16">
                <div className="w-20 h-20 rounded-2xl bg-gradient-to-br from-primary/10 to-accent/10 flex items-center justify-center mb-6">
                  <MessageSquare className="h-10 w-10 text-primary" />
                </div>
                <h3 className="text-xl font-semibold mb-2">开始跨库对话</h3>
                <p className="text-muted-foreground text-center mb-6 max-w-md">
                  从多个知识库中检索信息，获取综合性的智能回答。
                </p>
                <div className="flex gap-2 flex-wrap justify-center max-w-lg">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setInput('这些知识库主要讲了什么内容？')}
                    className="rounded-xl"
                  >
                    这些知识库主要讲了什么？
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setInput('有哪些关键概念和联系？')}
                    className="rounded-xl"
                  >
                    有哪些关键概念和联系？
                  </Button>
                </div>
              </div>
            ) : (
              messages.map((message) => (
                <div
                  key={message.id}
                  className={`flex ${message.role === 'user' ? 'justify-end' : 'justify-start'}`}
                >
                  <Card
                    className={`max-w-[80%] transition-all ${
                      message.role === 'user'
                        ? 'bg-gradient-to-br from-primary to-accent text-primary-foreground shadow-lg shadow-primary/25'
                        : 'glass border-border/50 shadow-md'
                    }`}
                  >
                    <CardContent className="p-4">
                      <p className="whitespace-pre-wrap">{message.content}</p>
                      {message.metadata?.kb_names && message.metadata.kb_names.length > 0 && (
                        <div className="mt-3 pt-3 border-t border-border/50 text-xs opacity-80">
                          <div className="flex items-center gap-2 mb-2">
                            <FileText className="h-3 w-3" />
                            <span>来源：{message.metadata.sources?.length || 0} 个文档</span>
                          </div>
                          <div className="flex flex-wrap gap-1">
                            {message.metadata.kb_names.slice(0, 5).map((kbName, i) => (
                              <span
                                key={i}
                                className="px-2 py-0.5 rounded-full bg-primary/10 text-primary text-xs"
                              >
                                {kbName}
                              </span>
                            ))}
                            {message.metadata.kb_names.length > 5 && (
                              <span className="px-2 py-0.5 text-xs text-muted-foreground">
                                +{message.metadata.kb_names.length - 5} 更多
                              </span>
                            )}
                          </div>
                        </div>
                      )}
                    </CardContent>
                  </Card>
                </div>
              ))
            )}
            {loading && (
              <div className="flex justify-start">
                <Card className="glass border-border/50 shadow-md">
                  <CardContent className="p-4">
                    <div className="flex gap-1">
                      <div className="w-2 h-2 bg-primary rounded-full animate-bounce" />
                      <div className="w-2 h-2 bg-primary rounded-full animate-bounce [animation-delay:0.2s]" />
                      <div className="w-2 h-2 bg-primary rounded-full animate-bounce [animation-delay:0.4s]" />
                    </div>
                  </CardContent>
                </Card>
              </div>
            )}
            <div ref={messagesEndRef} />
          </div>

          {/* Input */}
          <div className="flex-shrink-0 border-t border-border/50 p-4 glass bg-background/80">
            <div className="flex gap-3 max-w-4xl mx-auto">
              <Textarea
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder={
                  selectedKBs.size === 0
                    ? '请先选择至少一个知识库...'
                    : '输入您的问题，按 Enter 发送...'
                }
                className="min-h-[60px] resize-none rounded-xl bg-background/50 border-border/50 focus:border-primary/50 focus:ring-primary/20"
                disabled={loading || kbs.length === 0}
              />
              <Button
                onClick={sendMessage}
                disabled={loading || !input.trim() || selectedKBs.size === 0}
                className="self-end rounded-xl bg-gradient-to-r from-primary to-accent hover:from-primary/90 hover:to-accent/90 shadow-lg shadow-primary/25 min-w-[60px]"
              >
                <Send className="h-5 w-5" />
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
