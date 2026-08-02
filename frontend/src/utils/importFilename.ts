// Extract the "filename" label from a chosen import file.
// Rule: take the last underscore-delimited segment's first part.
// Strategy: strip extension, then split on the LAST underscore and return the
// part that comes BEFORE it (the "first part" after splitting at the last _).
// Examples:
//   chatgpt_pro_us_001.json -> "chatgpt_pro_us"
//   accounts.json           -> "accounts"  (no underscore: whole stem)
export function extractImportFilenamePart(fileName: string): string {
  if (!fileName) return ''
  // Strip the final extension (handles only the last dot).
  const slashIdx = fileName.lastIndexOf('/')
  const base = slashIdx >= 0 ? fileName.slice(slashIdx + 1) : fileName
  const dotIdx = base.lastIndexOf('.')
  const stem = dotIdx > 0 ? base.slice(0, dotIdx) : base

  const underIdx = stem.lastIndexOf('_')
  if (underIdx < 0) return stem
  return stem.slice(0, underIdx)
}
