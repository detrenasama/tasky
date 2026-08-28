// Форматирование длительностей (секунды → «Чч Мм» / «Мм Сс»).
export function fmtDuration(seconds: number): string {
  const s = Math.max(0, Math.floor(seconds))
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const sec = s % 60
  if (h > 0) return `${h}ч ${m}м`
  if (m > 0) return `${m}м ${sec}с`
  return `${sec}с`
}

// RFC3339 (или с миллисекундами) → локальную дату-время для отображения.
export function fmtDateTime(iso: string | null): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (isNaN(d.getTime())) return iso
  return d.toLocaleString('ru-RU', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function fmtDate(iso: string | null): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (isNaN(d.getTime())) return iso
  return d.toLocaleDateString('ru-RU')
}

// Текущее время в RFC3339 для отправки на сервер (начало/конец записей).
export function toRFC3339(d: Date): string {
  return d.toISOString()
}
