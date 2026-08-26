// Minimal type-safe i18n. Catalogs must stay key-compatible; enforced by tests.

export const en = {
  'app.title': 'ServerPanel',
  'nav.userPanel': 'User Panel',
  'nav.admin': 'Admin',
  'health.title': 'Server Health',
  'health.status': 'Status',
  'health.version': 'Version',
  'health.loading': 'Checking server…',
  'health.offline': 'API unreachable',
  'dash.welcomeUser': 'Welcome to your hosting panel',
  'dash.welcomeAdmin': 'Server administration console',
  'dash.comingSoon': 'Modules for this section are delivered in upcoming releases.',
} as const

export type MessageKey = keyof typeof en
