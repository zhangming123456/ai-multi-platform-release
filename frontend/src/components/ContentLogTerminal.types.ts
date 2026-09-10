export type LogLevel = 'info' | 'req' | 'ok' | 'err'

export interface ContentLogTerminalProps {
  logs: { id: string; time: string; level: LogLevel; message: string }[]
  isGenerating: boolean
}

export type ContentLogTerminalEmits = {
  (e: 'clear'): void
}
