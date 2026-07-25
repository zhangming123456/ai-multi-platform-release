import type { DirectiveBinding, ObjectDirective } from 'vue'
import { usePermissionStore } from '@/stores/permission'

interface PermBinding {
  key: string
  mode?: 'read' | 'write'
}

const vPerm: ObjectDirective<HTMLElement, PermBinding> = {
  mounted(el: HTMLElement, binding: DirectiveBinding<PermBinding>) {
    const permStore = usePermissionStore()
    const { key, mode = 'read' } = binding.value
    if (!permStore.hasPermission(key, mode)) {
      el.style.display = 'none'
    }
  },
  updated(el: HTMLElement, binding: DirectiveBinding<PermBinding>) {
    const permStore = usePermissionStore()
    const { key, mode = 'read' } = binding.value
    el.style.display = permStore.hasPermission(key, mode) ? '' : 'none'
  },
}

export default vPerm
