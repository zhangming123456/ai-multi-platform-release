export type PanelDrawEmits = {
  (e: 'style-change', color: string, width: number): void
  (e: 'clear'): void
}
