import type { DirectiveBinding, ObjectDirective } from 'vue'
import { usePermissionStore } from '@/stores/permission'

interface PermBinding {
  key: string
}

const vPerm: ObjectDirective<HTMLElement, PermBinding> = {
  mounted(el: HTMLElement, binding: DirectiveBinding<PermBinding>) {
    const permStore = usePermissionStore()
    if (!permStore.hasPermission(binding.value.key)) {
      el.style.display = 'none'
    }
  },
  updated(el: HTMLElement, binding: DirectiveBinding<PermBinding>) {
    const permStore = usePermissionStore()
    el.style.display = permStore.hasPermission(binding.value.key) ? '' : 'none'
  },
}

export default vPerm
