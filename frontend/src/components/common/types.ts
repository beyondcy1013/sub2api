/**
 * Common component types
 */

export interface Column {
  key: string
  label: string
  sortable?: boolean
  class?: string
  width?: string
  formatter?: (value: any, row: any) => string
}
