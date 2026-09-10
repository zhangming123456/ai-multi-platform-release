import type { Component } from 'vue'

export interface AppSidebarProps {
  collapsed: boolean
}

export type AppSidebarEmits = {
  toggle: []
  closeMobile: []
}

export interface SidebarGroupConfig {
  name: string
  icon: string
  order: number
  wrapGroup: boolean
}

export interface MenuEntry {
  key: string
  name: string
  path: string
  icon?: Component
  permKey?: string
}

export interface MenuGroup {
  key: string
  name: string
  icon?: Component
  children: MenuEntry[]
}

export type MenuItem = MenuEntry | MenuGroup
