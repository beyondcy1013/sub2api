export interface PelicanPromptScenario {
  id: string
  name: string
  prompt: string
}

export interface PelicanPromptScenarioState {
  scenarios: PelicanPromptScenario[]
  selectedId: string
}

interface StoredPelicanPromptScenarioState {
  selectedId?: unknown
  scenarios?: unknown
}

const STORAGE_KEY = 'sub2api:account-pelican-prompt-scenarios:v1'
const MAX_SCENARIOS = 20
const MAX_PROMPT_LENGTH = 20000

function isScenario(value: unknown): value is PelicanPromptScenario {
  if (!value || typeof value !== 'object') return false
  const item = value as Partial<PelicanPromptScenario>
  return (
    typeof item.id === 'string' &&
    item.id.length > 0 &&
    typeof item.name === 'string' &&
    item.name.trim().length > 0 &&
    typeof item.prompt === 'string' &&
    item.prompt.length <= MAX_PROMPT_LENGTH
  )
}

export function createPelicanPromptScenarios(
  defaultName: string,
  defaultPrompt: string
): PelicanPromptScenario[] {
  return [{ id: 'default', name: defaultName, prompt: defaultPrompt }]
}

export function loadPelicanPromptScenarios(
  defaultName: string,
  defaultPrompt: string
): PelicanPromptScenarioState {
  const fallback: PelicanPromptScenarioState = {
    scenarios: createPelicanPromptScenarios(defaultName, defaultPrompt),
    selectedId: 'default'
  }

  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return fallback

    const parsed = JSON.parse(raw) as StoredPelicanPromptScenarioState
    if (!parsed || !Array.isArray(parsed.scenarios)) return fallback

    const scenarios = parsed.scenarios
      .filter(isScenario)
      .slice(0, MAX_SCENARIOS)
      .map(item => ({
        id: item.id,
        name: item.name.trim(),
        prompt: item.prompt
      }))

    const uniqueScenarios = Array.from(
      new Map(scenarios.map(item => [item.id, item])).values()
    )
    if (uniqueScenarios.length === 0) return fallback

    const selectedId = typeof parsed.selectedId === 'string'
      && uniqueScenarios.some(item => item.id === parsed.selectedId)
      ? parsed.selectedId
      : uniqueScenarios[0].id

    return { scenarios: uniqueScenarios, selectedId }
  } catch {
    return fallback
  }
}

export function savePelicanPromptScenarios(state: PelicanPromptScenarioState): void {
  const scenarios = state.scenarios
    .filter(isScenario)
    .slice(0, MAX_SCENARIOS)
    .map(item => ({
      id: item.id,
      name: item.name.trim(),
      prompt: item.prompt
    }))

  if (scenarios.length === 0) return

  const selectedId = scenarios.some(item => item.id === state.selectedId)
    ? state.selectedId
    : scenarios[0].id

  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ scenarios, selectedId }))
  } catch {
    // Ignore storage failures such as private browsing quotas.
  }
}

export function pelicanPromptOptionLabel(prompt: string, maxLength = 28): string {
  const normalized = prompt.replace(/\s+/g, ' ').trim()
  if (normalized.length <= maxLength) return normalized
  return `${normalized.slice(0, maxLength)}...`
}
