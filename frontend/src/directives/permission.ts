import type { DirectiveBinding, ObjectDirective } from 'vue'
import { usePermissionStore } from '@/stores/permission'
import type { PermContext } from '@/stores/permission'

interface PermBinding {
  key: string
  ctx?: PermContext
}

const _permCache = new WeakMap<
  HTMLElement,
  { placeholder: Comment; originalParent: Node; originalNext: Node | null }
>()

function resolve(binding: DirectiveBinding<string | PermBinding>): {
  key: string
  ctx?: PermContext
} {
  const value = binding.value
  if (typeof value === 'string') {
    return { key: value }
  }
  return { key: value.key, ctx: value.ctx }
}

function removeEl(el: HTMLElement) {
  const parent = el.parentNode
  if (!parent) return
  const placeholder = document.createComment('v-perm')
  const next = el.nextSibling
  parent.insertBefore(placeholder, el)
  parent.removeChild(el)
  _permCache.set(el, { placeholder, originalParent: parent, originalNext: next })
}

function restoreEl(el: HTMLElement) {
  const cache = _permCache.get(el)
  if (!cache) return
  const { placeholder, originalParent, originalNext } = cache
  if (!placeholder.parentNode) {
    originalParent.insertBefore(el, originalNext)
  } else {
    placeholder.parentNode.insertBefore(el, placeholder)
    placeholder.parentNode.removeChild(placeholder)
  }
  _permCache.delete(el)
}

function checkAndApply(el: HTMLElement, binding: DirectiveBinding<string | PermBinding>) {
  const permStore = usePermissionStore()
  const { key, ctx } = resolve(binding)
  const has = permStore.hasPermission(key, ctx)

  if (!has && !_permCache.has(el)) {
    removeEl(el)
  } else if (has && _permCache.has(el)) {
    restoreEl(el)
  }
}

const vPerm: ObjectDirective<HTMLElement, string | PermBinding> = {
  mounted: checkAndApply,
  updated: checkAndApply,
  beforeUnmount(el: HTMLElement) {
    const cache = _permCache.get(el)
    if (cache) {
      // 1. 从 DOM 中移除占位注释节点
      if (cache.placeholder.parentNode) {
        cache.placeholder.parentNode.removeChild(cache.placeholder)
      }
      // 2. 清除 WeakMap（防止后续 restoreEl 误操作）
      _permCache.delete(el)
    }
  },
}

export default vPerm
