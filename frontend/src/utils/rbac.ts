const ADMIN_ONLY_RESOURCE_KEYS: ReadonlySet<string> = new Set([
  'db',
  'db_history',
  'api_docs',
  'permissions',
  'roles',
  'constraints',
  'notification',
])

function isAdminTypePermission(permKey: string): boolean {
  const firstSegment = permKey.split(':')[0]
  return ADMIN_ONLY_RESOURCE_KEYS.has(firstSegment)
}

function filterAdminOnlyPermissions<T extends { readKey?: string; writeKeys: string[] }>(
  items: T[],
): T[] {
  return items.filter((item) => {
    const checkKey = item.readKey || item.writeKeys[0]
    if (!checkKey) return false
    return !isAdminTypePermission(checkKey)
  })
}

export { ADMIN_ONLY_RESOURCE_KEYS, isAdminTypePermission, filterAdminOnlyPermissions }
