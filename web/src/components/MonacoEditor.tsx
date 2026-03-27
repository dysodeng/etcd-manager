import { useState, useEffect, useRef } from 'react'
import { Modal, Button, Tooltip } from 'antd'
import { ExpandOutlined, CompressOutlined } from '@ant-design/icons'
import Editor, { type EditorProps, type Monaco } from '@monaco-editor/react'
import type { editor } from 'monaco-editor'

function detectLanguage(value: string): string {
  const trimmed = value.trimStart()
  if (trimmed.startsWith('{') || trimmed.startsWith('[')) return 'json'
  if (/^---\s*$/m.test(trimmed) || /^\w[\w-]*\s*:/m.test(trimmed)) return 'yaml'
  if (trimmed.startsWith('<?xml') || trimmed.startsWith('<')) return 'xml'
  if (trimmed.startsWith('#!') || /^(export |source |if \[)/m.test(trimmed)) return 'shell'
  if (/^\[[\w.-]+\]\s*$/m.test(trimmed)) return 'ini'
  if (/^(server|location|upstream)\s*\{/m.test(trimmed)) return 'nginx'
  return 'plaintext'
}

interface Props {
  value: string
  onChange?: (value: string) => void
  language?: string
  height?: string | number
  readOnly?: boolean
}

export default function MonacoEditor({ value, onChange, language, height = 400, readOnly = false }: Props) {
  const lang = language ?? detectLanguage(value)
  const [expanded, setExpanded] = useState(false)
  const editorRef = useRef<editor.IStandaloneCodeEditor | null>(null)
  const monacoRef = useRef<Monaco | null>(null)

  useEffect(() => {
    if (editorRef.current && monacoRef.current) {
      const model = editorRef.current.getModel()
      if (model) {
        monacoRef.current.editor.setModelLanguage(model, lang)
      }
    }
  }, [lang])

  const handleMount = (ed: editor.IStandaloneCodeEditor, monaco: Monaco) => {
    editorRef.current = ed
    monacoRef.current = monaco
  }

  const options: EditorProps['options'] = {
    minimap: { enabled: false },
    readOnly,
    scrollBeyondLastLine: false,
    fontSize: 13,
    wordWrap: 'on',
    automaticLayout: true,
  }

  const expandedOptions: EditorProps['options'] = {
    ...options,
    minimap: { enabled: true },
    fontSize: 14,
  }

  return (
    <>
      <div style={{ position: 'relative' }}>
        <Editor
          height={height}
          language={lang}
          value={value}
          onChange={(v) => onChange?.(v ?? '')}
          options={options}
          theme="vs-dark"
          onMount={handleMount}
        />
        <Tooltip title="展开编辑器">
          <Button
            type="text"
            size="small"
            icon={<ExpandOutlined />}
            onClick={() => setExpanded(true)}
            style={{
              position: 'absolute',
              top: 4,
              right: 16,
              zIndex: 10,
              color: '#ccc',
              background: 'rgba(30,30,30,0.8)',
            }}
          />
        </Tooltip>
      </div>

      <Modal
        title={null}
        open={expanded}
        onCancel={() => setExpanded(false)}
        footer={null}
        width="90vw"
        style={{ top: 24 }}
        styles={{ body: { padding: 0 } }}
        closeIcon={
          <Tooltip title="收起编辑器">
            <CompressOutlined />
          </Tooltip>
        }
        destroyOnHidden
      >
        <Editor
          height="80vh"
          language={lang}
          value={value}
          onChange={(v) => onChange?.(v ?? '')}
          options={expandedOptions}
          theme="vs-dark"
          onMount={handleMount}
        />
      </Modal>
    </>
  )
}
