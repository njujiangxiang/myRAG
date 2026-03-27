import { useState, useEffect, useRef } from 'react'
import { useParams, Link } from 'react-router-dom'
import { ArrowLeft, Upload, FileText, CheckCircle, Clock, XCircle, Trash2 } from 'lucide-react'
import { useAuth } from '@/contexts/AuthContext'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'

interface Document {
  id: string
  filename: string
  file_size: number
  status: 'pending' | 'processing' | 'indexed' | 'error'
  created_at: string
  error_message?: string | null
}

const statusConfig = {
  pending: { icon: Clock, label: '等待中', color: 'text-yellow-500' },
  processing: { icon: Clock, label: '处理中', color: 'text-blue-500' },
  indexed: { icon: CheckCircle, label: '已索引', color: 'text-green-500' },
  error: { icon: XCircle, label: '错误', color: 'text-red-500' },
}

export default function KBDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { token } = useAuth()
  const [documents, setDocuments] = useState<Document[]>([])
  const [uploading, setUploading] = useState(false)
  const [uploadProgress, setUploadProgress] = useState(0)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const fetchDocuments = async () => {
    try {
      const response = await fetch(`/api/v1/kbs/${id}/docs`, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      })
      if (response.ok) {
        const data = await response.json()
        setDocuments(data)
      }
    } catch (error) {
      console.error('Failed to fetch documents:', error)
    }
  }

  useEffect(() => {
    fetchDocuments()
    // Poll for status updates
    const interval = setInterval(fetchDocuments, 5000)
    return () => clearInterval(interval)
  }, [id, token])

  const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file || !id) return

    setUploading(true)
    setUploadProgress(0)

    const formData = new FormData()
    formData.append('file', file)

    try {
      const response = await fetch(`/api/v1/kbs/${id}/docs`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
        },
        body: formData,
      })

      if (response.ok) {
        fetchDocuments()
      } else {
        const error = await response.json()
        alert(`上传失败：${error.error}`)
      }
    } catch (error) {
      console.error('Failed to upload:', error)
      alert('上传失败，请重试')
    } finally {
      setUploading(false)
      setUploadProgress(100)
      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
    }
  }

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
        fetchDocuments()
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

  return (
    <div className="p-6">
      {/* Header */}
      <div className="flex items-center gap-4 mb-8">
        <Link to="/kbs">
          <Button variant="ghost" size="icon" className="rounded-xl hover:bg-secondary/50">
            <ArrowLeft className="h-5 w-5" />
          </Button>
        </Link>
        <div>
          <h1 className="text-3xl font-bold bg-gradient-to-r from-primary to-accent bg-clip-text text-transparent">
            知识库详情
          </h1>
          <p className="text-muted-foreground mt-1">管理文档和查看处理状态</p>
        </div>
      </div>

      {/* Upload Area */}
      <Card className="mb-8 glass border-border/50 shadow-xl">
        <CardContent className="pt-6">
          <div
            className={`border-2 border-dashed rounded-2xl p-10 text-center transition-all ${
              uploading
                ? 'bg-muted/50 border-primary/50'
                : 'hover:bg-secondary/50 hover:border-primary/50 cursor-pointer'
            }`}
            onClick={() => !uploading && fileInputRef.current?.click()}
          >
            <input
              ref={fileInputRef}
              type="file"
              className="hidden"
              accept=".pdf,.doc,.docx,.md,.txt,.csv,.xls,.xlsx"
              onChange={handleFileSelect}
              disabled={uploading}
            />
            <div className={`w-16 h-16 rounded-2xl mx-auto mb-4 flex items-center justify-center transition-all ${
              uploading
                ? 'bg-gradient-to-br from-primary/90 to-accent/90'
                : 'bg-gradient-to-br from-primary/10 to-accent/10'
            }`}>
              <Upload className={`h-8 w-8 ${
                uploading
                  ? 'text-white animate-pulse'
                  : 'text-primary'
              }`} />
            </div>
            {uploading ? (
              <div>
                <p className="font-semibold text-lg mb-3">上传中...</p>
                <Progress value={uploadProgress} className="w-full max-w-xs mx-auto h-2" />
                <p className="text-sm text-muted-foreground mt-2">{uploadProgress}%</p>
              </div>
            ) : (
              <div>
                <p className="font-semibold text-lg mb-2">拖拽文件到此处，或点击浏览</p>
                <p className="text-sm text-muted-foreground">
                  支持 PDF, DOCX, MD, TXT, CSV, XLSX 格式（最大 50MB）
                </p>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Documents List */}
      <Card className="glass border-border/50 shadow-xl">
        <CardHeader>
          <CardTitle className="flex items-center gap-3">
            <div className="w-8 h-8 rounded-xl bg-gradient-to-br from-primary/10 to-accent/10 flex items-center justify-center">
              <FileText className="h-4 w-4 text-primary" />
            </div>
            文档列表
            <span className="text-sm font-normal text-muted-foreground ml-2">
              {documents.length} 个文档
            </span>
          </CardTitle>
        </CardHeader>
        <CardContent>
          {documents.length === 0 ? (
            <div className="text-center py-12">
              <div className="w-16 h-16 rounded-2xl bg-muted flex items-center justify-center mx-auto mb-4">
                <FileText className="h-8 w-8 text-muted-foreground opacity-50" />
              </div>
              <p className="font-semibold text-lg mb-2">暂无文档</p>
              <p className="text-sm text-muted-foreground">上传第一个文档开始使用</p>
            </div>
          ) : (
            <div className="space-y-3">
              {documents.map((doc) => {
                const StatusIcon = statusConfig[doc.status].icon
                const statusColor = statusConfig[doc.status].color
                return (
                  <div
                    key={doc.id}
                    className="flex items-center gap-4 p-4 rounded-xl border border-border/50 hover:bg-secondary/50 transition-all group"
                  >
                    <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-primary/10 to-accent/10 flex items-center justify-center flex-shrink-0">
                      <FileText className="h-5 w-5 text-primary" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="font-medium truncate">{doc.filename}</p>
                      <p className="text-sm text-muted-foreground">
                        {formatFileSize(doc.file_size)} · {formatDate(doc.created_at)}
                      </p>
                    </div>
                    <div className={`flex items-center gap-2 px-3 py-1.5 rounded-full bg-secondary/50 ${statusColor}`}>
                      <StatusIcon className="h-4 w-4" />
                      <span className="text-sm font-medium">{statusConfig[doc.status].label}</span>
                    </div>
                    {doc.error_message && (
                      <p className="text-sm text-destructive max-w-xs truncate bg-destructive/10 px-3 py-1.5 rounded-full">
                        {doc.error_message}
                      </p>
                    )}
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => deleteDocument(doc.id)}
                      className="opacity-0 group-hover:opacity-100 transition-opacity rounded-xl hover:bg-destructive/10 hover:text-destructive"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
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
