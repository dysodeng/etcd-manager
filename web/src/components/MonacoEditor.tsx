import Editor, { type EditorProps } from '@monaco-editor/react'

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

  const options: EditorProps['options'] = {
    minimap: { enabled: false },
    readOnly,
    scrollBeyondLastLine: false,
    fontSize: 13,
    wordWrap: 'on',
    automaticLayout: true,
  }

  return (
    <Editor
      height={height}
      language={lang}
      value={value}
      onChange={(v) => onChange?.(v ?? '')}
      options={options}
      theme="vs-dark"
    />
  )
}
