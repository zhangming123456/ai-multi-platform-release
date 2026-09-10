import type { Component } from 'vue'

export interface StatCardProps {
  title: string
  value: number
  trend: string
  icon: Component
  color: string
  delay?: number
}
