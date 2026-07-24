import type { Directive, DirectiveBinding } from 'vue'
import { usePermissionStore } from '@/stores/permission'

const vPerm: Directive = {
  mounted(el: HTMLElement, binding: DirectiveBinding) {
    const { value } = binding
    if (!value) return

    const permStore = usePermissionStore()
    let hasAccess = false

    if (typeof value === 'string') {
      hasAccess = permStore.hasPermission(value, 'read')
    } else if (Array.isArray(value)) {
      hasAccess = value.some((key: string) => permStore.hasPermission(key, 'read'))
    } else if (typeof value === 'object' && value !== null) {
      const { key, mode } = value
      hasAccess = permStore.hasPermission(key, mode || 'read')
    }

    if (!hasAccess) {
      el.parentNode?.removeChild(el)
    }
  },
}

export default vPerm
