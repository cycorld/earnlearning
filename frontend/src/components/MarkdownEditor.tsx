import { useRef, useState, useCallback, useEffect } from 'react'
import { api } from '@/lib/api'
import { Textarea } from '@/components/ui/textarea'
import { Button } from '@/components/ui/button'
import { MarkdownContent } from './MarkdownContent'
import {
  Bold,
  Italic,
  List,
  ListOrdered,
  Code,
  Link,
  ImagePlus,
  Paperclip,
  Eye,
  Edit3,
  Loader2,
  X,
} from 'lucide-react'
import { toast } from 'sonner'

interface UploadedFile {
  id: number
  url: string
  filename: string
  mime_type: string
  size: number
}

interface MarkdownEditorProps {
  value: string
  onChange: (value: string) => void
  placeholder?: string
  rows?: number
  compact?: boolean
  onFilesChange?: (files: UploadedFile[]) => void
}

const formatBytes = (bytes: number) => {
  if (bytes < 1024) return `${bytes}B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)}MB`
}

export function MarkdownEditor({
  value,
  onChange,
  placeholder = '마크다운으로 작성하세요...',
  rows = 5,
  compact = false,
  onFilesChange,
}: MarkdownEditorProps) {
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [preview, setPreview] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [uploadedFiles, setUploadedFiles] = useState<UploadedFile[]>([])

  // Clear uploaded files when value is reset externally (e.g. after comment submit)
  useEffect(() => {
    if (value === '' && uploadedFiles.length > 0) {
      setUploadedFiles([])
    }
  }, [value]) // eslint-disable-line react-hooks/exhaustive-deps

  const insertText = useCallback(
    (before: string, after: string = '') => {
      const textarea = textareaRef.current
      if (!textarea) return

      const start = textarea.selectionStart
      const end = textarea.selectionEnd
      const selected = value.substring(start, end)
      const newText =
        value.substring(0, start) +
        before +
        selected +
        after +
        value.substring(end)
      onChange(newText)

      requestAnimationFrame(() => {
        textarea.focus()
        const cursorPos = start + before.length + selected.length
        textarea.setSelectionRange(cursorPos, cursorPos)
      })
    },
    [value, onChange],
  )

  const handleUpload = useCallback(
    async (files: FileList | null) => {
      if (!files || files.length === 0) return

      setUploading(true)
      const newFiles: UploadedFile[] = []

      for (const file of Array.from(files)) {
        if (file.size > 10 * 1024 * 1024) {
          toast.error(`${file.name}: 파일 크기는 10MB 이하여야 합니다.`)
          continue
        }

        try {
          const formData = new FormData()
          formData.append('file', file)
          const result = await api.post<UploadedFile>('/upload', formData)
          newFiles.push(result)

          // Auto-insert image or file link into editor
          if (file.type.startsWith('image/')) {
            insertText(`![${file.name}](${result.url})\n`)
          } else {
            insertText(`[📎 ${file.name}](${result.url})\n`)
          }
        } catch {
          toast.error(`${file.name} 업로드에 실패했습니다.`)
        }
      }

      if (newFiles.length > 0) {
        const updated = [...uploadedFiles, ...newFiles]
        setUploadedFiles(updated)
        onFilesChange?.(updated)
      }
      setUploading(false)

      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
    },
    [insertText, uploadedFiles, onFilesChange],
  )

  const removeFile = useCallback(
    (fileToRemove: UploadedFile) => {
      const updated = uploadedFiles.filter((f) => f.id !== fileToRemove.id)
      setUploadedFiles(updated)
      onFilesChange?.(updated)
    },
    [uploadedFiles, onFilesChange],
  )

  const handlePaste = useCallback(
    (e: React.ClipboardEvent) => {
      const items = e.clipboardData.items
      const imageFiles: File[] = []
      for (const item of Array.from(items)) {
        if (item.type.startsWith('image/')) {
          const file = item.getAsFile()
          if (file) imageFiles.push(file)
        }
      }
      if (imageFiles.length > 0) {
        e.preventDefault()
        const dt = new DataTransfer()
        imageFiles.forEach((f) => dt.items.add(f))
        handleUpload(dt.files)
      }
    },
    [handleUpload],
  )

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault()
      handleUpload(e.dataTransfer.files)
    },
    [handleUpload],
  )

  const toolbarButtons = compact
    ? [
        { icon: Bold, action: () => insertText('**', '**'), title: '굵게' },
        { icon: Italic, action: () => insertText('*', '*'), title: '기울임' },
        { icon: Code, action: () => insertText('`', '`'), title: '코드' },
      ]
    : [
        { icon: Bold, action: () => insertText('**', '**'), title: '굵게' },
        { icon: Italic, action: () => insertText('*', '*'), title: '기울임' },
        { icon: Code, action: () => insertText('`', '`'), title: '코드' },
        {
          icon: List,
          action: () => insertText('\n- '),
          title: '목록',
        },
        {
          icon: ListOrdered,
          action: () => insertText('\n1. '),
          title: '번호 목록',
        },
        {
          icon: Link,
          action: () => insertText('[', '](url)'),
          title: '링크',
        },
      ]

  return (
    <div className="space-y-1">
      {/* Toolbar */}
      <div className="flex items-center gap-0.5 rounded-t-md border border-b-0 bg-muted/50 px-1 py-0.5">
        {toolbarButtons.map(({ icon: Icon, action, title }) => (
          <Button
            key={title}
            type="button"
            variant="ghost"
            size="sm"
            className="h-7 w-7 p-0"
            onClick={action}
            title={title}
          >
            <Icon className="h-3.5 w-3.5" />
          </Button>
        ))}
        <div className="mx-1 h-4 w-px bg-border" />
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-7 w-7 p-0"
          onClick={() => fileInputRef.current?.click()}
          disabled={uploading}
          title="이미지 첨부"
        >
          {uploading ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <ImagePlus className="h-3.5 w-3.5" />
          )}
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-7 w-7 p-0"
          onClick={() => fileInputRef.current?.click()}
          disabled={uploading}
          title="파일 첨부"
        >
          <Paperclip className="h-3.5 w-3.5" />
        </Button>
        <div className="flex-1" />
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-7 gap-1 px-2 text-xs"
          onClick={() => setPreview(!preview)}
        >
          {preview ? (
            <>
              <Edit3 className="h-3 w-3" /> 편집
            </>
          ) : (
            <>
              <Eye className="h-3 w-3" /> 미리보기
            </>
          )}
        </Button>
      </div>

      {/* Editor or Preview */}
      {preview ? (
        <div className="min-h-[80px] rounded-b-md border p-3">
          {value.trim() ? (
            <MarkdownContent content={value} />
          ) : (
            <p className="text-sm text-muted-foreground">미리볼 내용이 없습니다.</p>
          )}
        </div>
      ) : (
        <Textarea
          ref={textareaRef}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          rows={rows}
          className="rounded-t-none border-t-0 font-mono text-sm"
          onPaste={handlePaste}
          onDrop={handleDrop}
          onDragOver={(e) => e.preventDefault()}
        />
      )}

      {/* Hidden file input */}
      <input
        ref={fileInputRef}
        type="file"
        className="hidden"
        multiple
        accept="image/*,.pdf,.doc,.docx,.xls,.xlsx,.pptx,.zip,.txt"
        onChange={(e) => handleUpload(e.target.files)}
      />

      {/* Uploaded files list */}
      {uploadedFiles.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {uploadedFiles.map((file) => (
            <div
              key={file.id}
              className="flex items-center gap-1 rounded-md bg-muted px-2 py-1 text-xs"
            >
              {file.mime_type.startsWith('image/') ? (
                <img
                  src={file.url}
                  alt={file.filename}
                  className="h-6 w-6 rounded object-cover"
                />
              ) : (
                <Paperclip className="h-3 w-3" />
              )}
              <span className="max-w-[120px] truncate">{file.filename}</span>
              <span className="text-muted-foreground">
                ({formatBytes(file.size)})
              </span>
              <button
                type="button"
                onClick={() => removeFile(file)}
                className="ml-0.5 rounded-full p-0.5 hover:bg-destructive/20"
              >
                <X className="h-3 w-3" />
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
