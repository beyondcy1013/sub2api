/**
 * Common component types
 */

export interface Column {
  key: string
  label: string
  sortable?: boolean
  class?: string
  width?: string
  /** Keep the declared width even when dynamicColumnWidths is enabled. */
  fixedWidth?: boolean
  formatter?: (value: any, row: any) => string
}
