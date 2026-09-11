import type { ConditionSqlColumnMap } from './ConditionBuilder.types'

export const CONDITION_SQL_TABLE = 'contents'

export const CONDITION_SQL_ORDER_BY = 'created_at DESC'

export const CONDITION_SQL_LIMIT = 100

export const CONDITION_SQL_COLUMN_MAP: ConditionSqlColumnMap[] = [
  { variable: 'title', table: 'contents', column: 'title', queryable: true },
  { variable: 'body', table: 'contents', column: 'body', queryable: true },
  { variable: 'platform', table: 'contents', column: 'platform', queryable: true },
  { variable: 'status', table: 'contents', column: 'status', queryable: true },
  { variable: 'ai_generated', table: 'contents', column: 'ai_generated', queryable: true },
  { variable: 'created_at', table: 'contents', column: 'created_at', queryable: true },
  { variable: 'updated_at', table: 'contents', column: 'updated_at', queryable: true },
  { variable: 'fans_count', table: '', column: '', queryable: false },
  { variable: 'like_count', table: '', column: '', queryable: false },
  { variable: 'comment_count', table: '', column: '', queryable: false },
  { variable: 'is_verified', table: '', column: '', queryable: false },
  { variable: 'account_level', table: '', column: '', queryable: false },
  { variable: 'published_at', table: '', column: '', queryable: false },
  { variable: 'publish_slot', table: '', column: '', queryable: false },
]

export function conditionSqlColumnIndex(
  columnMap: ConditionSqlColumnMap[] = CONDITION_SQL_COLUMN_MAP,
): Map<string, ConditionSqlColumnMap> {
  return new Map(columnMap.map((entry) => [entry.variable, entry]))
}

export function conditionSqlColumnOf(
  variable: string,
  columnMap: ConditionSqlColumnMap[] = CONDITION_SQL_COLUMN_MAP,
): ConditionSqlColumnMap | undefined {
  return conditionSqlColumnIndex(columnMap).get(variable)
}

export function isConditionSqlQueryable(
  variable: string,
  columnMap: ConditionSqlColumnMap[] = CONDITION_SQL_COLUMN_MAP,
): boolean {
  return conditionSqlColumnIndex(columnMap).get(variable)?.queryable === true
}

export function conditionSqlSelectColumns(
  variables: string[],
  columnMap: ConditionSqlColumnMap[] = CONDITION_SQL_COLUMN_MAP,
): string[] {
  const index = conditionSqlColumnIndex(columnMap)
  const seen = new Set<string>()
  const columns: string[] = []
  for (const variable of variables) {
    const entry = index.get(variable)
    if (!entry || !entry.queryable || !entry.column) continue
    if (seen.has(entry.column)) continue
    seen.add(entry.column)
    columns.push(entry.column)
  }
  return columns
}
