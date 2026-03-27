import { useState, useEffect, useRef } from 'react'
import { useParams, Link } from 'react-router-dom'
import { ArrowLeft, Send, MessageSquare, FileText, ChevronDown, ChevronUp, ExternalLink, Trash2, History, PanelLeftClose, PanelLeftOpen } from 'lucide-react'
import { useAuth } from '@/contexts/AuthContext'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Card, CardContent } from '@/components/ui/card'

interface Message {
  id: string
  role: 'user' | 'assistant'
  content: string
  metadata?: {
    sources?: string[]
    context_chunks?: number
  }
}

interface Document {
  id: string
  filename: string
  file_size: number
  status: 'pending' | 'processing' | 'indexed' | 'error'
}

interface CitationDoc {
  id: string
  filename: string
  indexed: boolean
}

interface ChatSession {
  id: string
  kb_id: string
  user_id: string
  title: string | null
  created_at: string
  updated_at: string
}

export default function ChatPage() {
  const { id } = useParams<{ id: string }>()
  const { token } = useAuth()
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [documentCache, setDocumentCache] = useState<Map<string, Document>>(new Map())
  const [expandedMessages, setExpandedMessages] = useState<Set<string>>(new Set())
  const [sessions, setSessions] = useState<ChatSession[]>([])
  const [currentSessionId, setCurrentSessionId] = useState<string | null>(null)
  const [showSessions, setShowSessions] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  useEffect(() => {
    scrollToBottom()
  }, [messages])

  useEffect(() => {
    if (id) {
      fetchSessions()
    }
  }, [id, token])

  const fetchSessions = async () => {
    try {
      const response = await fetch(`/api/v1/kbs/${id}/sessions`, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      })
      if (response.ok) {
        const data = await response.json()
        setSessions(data)
      }
    } catch (error) {
      console.error('Failed to fetch sessions:', error)
    }
  }

  const loadSessionMessages = async (sessionId: string) => {
    try {
      const response = await fetch(`/api/v1/sessions/${sessionId}/messages`, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      })
      if (response.ok) {
        const data = await response.json()
        setMessages(data.map((msg: any) => ({
          id: msg.id,
          role: msg.role,
          content: msg.content,
          metadata: msg.metadata,
        })))
        setCurrentSessionId(sessionId)

        // Fetch document details for citations
        const docIds = data
          .filter((m: any) => m.metadata?.sources)
          .flatMap((m: any) => m.metadata.sources)
        if (docIds.length > 0) {
          fetchDocumentDetails(docIds)
        }
      }
    } catch (error) {
      console.error('Failed to load session messages:', error)
    }
  }

  const deleteSession = async (sessionId: string, e: React.MouseEvent) => {
    e.stopPropagation()
    if (!confirm('确定要删除这个会话吗？')) return

    try {
      const response = await fetch(`/api/v1/sessions/${sessionId}`, {
        method: 'DELETE',
        headers: {
          Authorization: `Bearer ${token}`,
        },
      })
      if (response.ok) {
        setSessions(prev => prev.filter(s => s.id !== sessionId))
        if (currentSessionId === sessionId) {
          setMessages([])
          setCurrentSessionId(null)
        }
      }
    } catch (error) {
      console.error('Failed to delete session:', error)
    }
  }

  // Fetch document details for citations
  const fetchDocumentDetails = async (docIds: string[]) => {
    const uncachedIds = docIds.filter(id => !documentCache.has(id))
    if (uncachedIds.length === 0) return

    try {
      const promises = uncachedIds.map(async (docId) => {
        const response = await fetch(`/api/v1/docs/${docId}`, {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        })
        if (response.ok) {
          return response.json()
        }
        return null
      })

      const results = await Promise.all(promises)
      setDocumentCache(prev => {
        const newCache = new Map(prev)
        results.forEach((doc: Document | null) => {
          if (doc) {
            newCache.set(doc.id, doc)
          }
        })
        return newCache
      })
    } catch (error) {
      console.error('Failed to fetch document details:', error)
    }
  }

  const sendMessage = async () => {
    if (!input.trim() || !id) return

    const userMessage: Message = {
      id: Date.now().toString(),
      role: 'user',
      content: input,
    }

    setMessages((prev) => [...prev, userMessage])
    setInput('')
    setLoading(true)

    try {
      const response = await fetch(`/api/v1/kbs/${id}/chat`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ content: input }),
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
      setCurrentSessionId(null) // New conversation, not linked to a session

      // Fetch document details for citations
      if (data.metadata?.sources && data.metadata.sources.length > 0) {
        await fetchDocumentDetails(data.metadata.sources)
      }

      // Refresh sessions list
      fetchSessions()
    } catch (error) {
      console.error('Failed to send message:', error)
      setMessages((prev) => [
        ...prev,
        {
          id: 'error',
          role: 'assistant',
          content: '抱歉，发生错误。请稍后重试。',
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

  const toggleCitations = (messageId: string) => {
    setExpandedMessages(prev => {
      const next = new Set(prev)
      if (next.has(messageId)) {
        next.delete(messageId)
      } else {
        next.add(messageId)
      }
      return next
    })
  }

  const getCitationDocs = (sources?: string[]): CitationDoc[] => {
    if (!sources) return []
    return sources.map(id => {
      const doc = documentCache.get(id)
      return {
        id,
        filename: doc?.filename || 'Unknown Document',
        indexed: doc?.status === 'indexed',
      }
    })
  }

  const formatSessionTitle = (session: ChatSession) => {
    if (session.title) return session.title
    const date = new Date(session.created_at)
    return `会话 ${date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })}`
  }

  return (
    <div className="flex flex-col h-[calc(100vh-4rem)]">
      {/* Header */}
      <div className="border-b border-border/50 p-4 flex items-center gap-4 glass">
        <Link to={`/kbs/${id}`}>
          <Button variant="ghost" size="icon" className="rounded-xl hover:bg-secondary/50">
            <ArrowLeft className="h-5 w-5" />
          </Button>
        </Link>
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 rounded-xl bg-gradient-to-br from-primary/90 to-accent/90 flex items-center justify-center">
              <MessageSquare className="h-4 w-4 text-white" />
            </div>
            <div>
              <h2 className="font-semibold bg-gradient-to-r from-primary to-accent bg-clip-text text-transparent">
                智能助手
              </h2>
              <p className="text-xs text-muted-foreground">基于知识库内容回答问题</p>
            </div>
          </div>
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setShowSessions(!showSessions)}
          className="rounded-xl hover:bg-secondary/50"
        >
          {showSessions ? <PanelLeftClose className="h-4 w-4" /> : <PanelLeftOpen className="h-4 w-4" />}
        </Button>
      </div>

      <div className="flex flex-1 overflow-hidden">
        {/* Sessions Sidebar */}
        {showSessions && (
          <div className="w-64 border-r border-border/50 bg-background/50 overflow-y-auto">
            <div className="p-3">
              <div className="flex items-center gap-2 mb-3 px-2">
                <History className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm font-medium text-muted-foreground">会话历史</span>
              </div>
              {sessions.length === 0 ? (
                <p className="text-xs text-muted-foreground px-2 py-4 text-center">暂无会话历史</p>
              ) : (
                <div className="space-y-1">
                  {sessions.map((session) => (
                    <div
                      key={session.id}
                      className={`group flex items-center gap-2 p-2 rounded-lg cursor-pointer transition-colors ${
                        currentSessionId === session.id
                          ? 'bg-primary/10 border border-primary/20'
                          : 'hover:bg-secondary/50 border border-transparent'
                      }`}
                      onClick={() => loadSessionMessages(session.id)}
                    >
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium truncate">{formatSessionTitle(session)}</p>
                        <p className="text-xs text-muted-foreground">
                          {new Date(session.updated_at).toLocaleString('zh-CN', {
                            month: 'short',
                            day: 'numeric',
                            hour: '2-digit',
                            minute: '2-digit',
                          })}
                        </p>
                      </div>
                      <button
                        onClick={(e) => deleteSession(session.id, e)}
                        className="opacity-0 group-hover:opacity-100 p-1 hover:bg-destructive/10 rounded transition-opacity"
                      >
                        <Trash2 className="h-3.5 w-3.5 text-destructive" />
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}

        {/* Messages Area */}
        <div className="flex-1 flex flex-col overflow-hidden">
          <div className="flex-1 overflow-y-auto p-4 space-y-4">
            {messages.length === 0 ? (
              <Card className="glass border-border/50 shadow-xl h-full">
                <CardContent className="flex flex-col items-center justify-center py-16">
                  <div className="w-20 h-20 rounded-2xl bg-gradient-to-br from-primary/10 to-accent/10 flex items-center justify-center mb-6">
                    <MessageSquare className="h-10 w-10 text-primary" />
                  </div>
                  <h3 className="text-xl font-semibold mb-2">开始对话</h3>
                  <p className="text-muted-foreground text-center mb-6 max-w-md">
                    提问关于知识库内容的问题，我会帮您查找相关信息并给出答案。
                  </p>
                  <div className="flex gap-2 flex-wrap justify-center max-w-lg">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setInput('知识库主要讲了什么内容？')}
                      className="rounded-xl"
                    >
                      知识库主要讲了什么？
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setInput('有哪些关键概念？')}
                      className="rounded-xl"
                    >
                      有哪些关键概念？
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ) : (
              messages.map((message) => (
                <div
                  key={message.id}
                  className={`flex ${message.role === 'user' ? 'justify-end' : 'justify-start'}`}
                >
                  <div className="max-w-[80%]">
                    <Card
                      className={`transition-all ${
                        message.role === 'user'
                          ? 'bg-gradient-to-br from-primary to-accent text-primary-foreground shadow-lg shadow-primary/25'
                          : 'glass border-border/50 shadow-md'
                      }`}
                    >
                      <CardContent className="p-4">
                        <p className="whitespace-pre-wrap">{message.content}</p>
                        {message.metadata?.sources && message.metadata.sources.length > 0 && (
                          <div className="mt-3">
                            <div
                              className={`flex items-center gap-2 text-xs opacity-80 cursor-pointer hover:opacity-100 transition-opacity ${
                                message.role === 'user' ? 'text-primary-foreground' : 'text-muted-foreground'
                              }`}
                              onClick={() => toggleCitations(message.id)}
                            >
                              <FileText className="h-3 w-3" />
                              <span>{message.metadata.sources.length} 个来源</span>
                              {expandedMessages.has(message.id) ? (
                                <ChevronUp className="h-3 w-3" />
                              ) : (
                                <ChevronDown className="h-3 w-3" />
                              )}
                            </div>
                            {expandedMessages.has(message.id) && (
                              <div className={`mt-2 space-y-1.5 text-xs ${
                                message.role === 'user' ? 'text-primary-foreground' : 'text-muted-foreground'
                              }`}>
                                {getCitationDocs(message.metadata.sources).map((doc, idx) => (
                                  <div
                                    key={doc.id}
                                    className={`flex items-center gap-2 p-2 rounded-lg ${
                                      message.role === 'user'
                                        ? 'bg-white/10'
                                        : 'bg-secondary/50'
                                    }`}
                                  >
                                    <div className={`flex-1 truncate ${
                                      doc.indexed ? '' : 'opacity-60'
                                    }`}>
                                      <span className="font-medium">[{idx + 1}] {doc.filename}</span>
                                      {!doc.indexed && (
                                        <span className="ml-2 px-1.5 py-0.5 rounded-full bg-yellow-500/20 text-yellow-600 text-[10px]">
                                          未索引
                                        </span>
                                      )}
                                    </div>
                                    <Link
                                      to={`/kbs/${id}/docs/${doc.id}`}
                                      className={`shrink-0 ${
                                        message.role === 'user'
                                          ? 'hover:text-primary-foreground'
                                          : 'hover:text-primary'
                                      }`}
                                    >
                                      <ExternalLink className="h-3 w-3" />
                                    </Link>
                                  </div>
                                ))}
                              </div>
                            )}
                          </div>
                        )}
                      </CardContent>
                    </Card>
                  </div>
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
          <div className="border-t border-border/50 p-4 glass">
            <div className="flex gap-3 max-w-4xl mx-auto">
              <Textarea
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder="输入您的问题..."
                className="min-h-[60px] resize-none rounded-xl bg-background/50 border-border/50 focus:border-primary/50 focus:ring-primary/20"
                disabled={loading}
              />
              <Button
                onClick={sendMessage}
                disabled={loading || !input.trim()}
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
