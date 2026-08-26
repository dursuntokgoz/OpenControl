import type { en } from './en'

// Turkish catalog. Keys must mirror `en` exactly (checked by unit test).
export const tr: Record<keyof typeof en, string> = {
  'app.title': 'ServerPanel',
  'nav.userPanel': 'Kullanıcı Paneli',
  'nav.admin': 'Yönetim',
  'health.title': 'Sunucu Durumu',
  'health.status': 'Durum',
  'health.version': 'Sürüm',
  'health.loading': 'Sunucu kontrol ediliyor…',
  'health.offline': "API'ye erişilemiyor",
  'dash.welcomeUser': 'Hosting panelinize hoş geldiniz',
  'dash.welcomeAdmin': 'Sunucu yönetim konsolu',
  'dash.comingSoon': 'Bu bölümün modülleri yaklaşan sürümlerde teslim edilecek.',
}
