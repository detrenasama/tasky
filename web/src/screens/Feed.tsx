import { useEffect, useState } from 'react'
import { api } from '../api'
import type { FeedItem } from '../types/feed'
import { fmtDate } from '../fmt'

export default function Feed({ onError }: { onError?: (m: string) => void }) {
  const [items, setItems] = useState<FeedItem[]>([])
  const [kind, setKind] = useState('')
  const [unreadOnly, setUnreadOnly] = useState(false)
  const [archived, setArchived] = useState(false)
  const [selected, setSelected] = useState<FeedItem | null>(null)

  const load = async () => {
    try {
      const list = await api.listFeed({ kind: kind || undefined, unread_only: unreadOnly || undefined, archived: archived || undefined })
      setItems(list)
      if (list.length > 0) setSelected(list[0])
    } catch (e: any) {
      onError?.(String(e))
    }
  }

  useEffect(() => { load() }, [kind, unreadOnly, archived])

  const markRead = async (it: FeedItem) => {
    if (it.is_read) return
    try {
      await api.markFeedRead(it.id)
      setItems((prev) => prev.map((x) => (x.id === it.id ? { ...x, is_read: true } : x)))
    } catch {}
  }

  useEffect(() => {
    if (selected && !selected.is_read) markRead(selected)
  }, [selected?.id])

  const grouped = groupByDay(items)

  return (
    <div style={{ display: 'flex', gap: 16 }}>
      <div style={{ flex: '0 0 45%', border: '1px solid #333', padding: 8, minHeight: 400, overflow: 'auto' }}>
        <div style={{ display: 'flex', gap: 8, marginBottom: 8, flexWrap: 'wrap' }}>
          <select value={kind} onChange={(e) => setKind(e.target.value)}>
            <option value="">все категории</option>
            <option value="overdue">просрочено</option>
            <option value="due_today">срок сегодня</option>
            <option value="due_tomorrow">срок завтра</option>
            <option value="due_week">срок неделя</option>
            <option value="scheduled_today">запланировано</option>
            <option value="update">обновления</option>
          </select>
          <label><input type="checkbox" checked={unreadOnly} onChange={(e) => setUnreadOnly(e.target.checked)} /> непрочитанные</label>
          <label><input type="checkbox" checked={archived} onChange={(e) => setArchived(e.target.checked)} /> архив</label>
          {archived && <button onClick={async () => { await api.deleteArchivedFeed(); load() }}>удалить архив</button>}
        </div>
        {Object.entries(grouped).map(([day, list]) => (
          <div key={day}>
            <div style={{ color: '#888', fontSize: 12, margin: '8px 0 4px' }}>{day}</div>
            {list.map((it) => (
              <div
                key={it.id}
                onClick={() => setSelected(it)}
                style={{
                  padding: '6px 8px',
                  background: selected?.id === it.id ? '#333' : it.is_read ? '#222' : '#1a1a1a',
                  borderLeft: it.is_read ? '2px solid transparent' : '2px solid #d33',
                  cursor: 'pointer',
                  marginBottom: 2,
                }}
              >
                <span style={{ color: it.is_read ? '#888' : '#fff' }}>{!it.is_read ? '● ' : ''}{it.title}</span>
                <span style={{ color: '#888', fontSize: 12 }}> — {it.kind}</span>
              </div>
            ))}
          </div>
        ))}
        {items.length === 0 && <div style={{ color: '#888' }}>Лента пуста.</div>}
      </div>
      <div style={{ flex: 1, border: '1px solid #333', padding: 12 }}>
        {selected ? (
          <>
            <h3>{selected.title}</h3>
            <div style={{ color: '#888', fontSize: 12 }}>{selected.kind} · {fmtDate(selected.created_at)}</div>
            <p style={{ whiteSpace: 'pre-wrap' }}>{selected.body}</p>
          </>
        ) : (
          <div style={{ color: '#888' }}>Выберите запись.</div>
        )}
      </div>
    </div>
  )
}

function groupByDay(items: FeedItem[]): Record<string, FeedItem[]> {
  const out: Record<string, FeedItem[]> = {}
  for (const it of items) {
    const d = new Date(it.created_at)
    const label = dayLabel(d)
    if (!out[label]) out[label] = []
    out[label].push(it)
  }
  return out
}

function dayLabel(d: Date): string {
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const day = new Date(d.getFullYear(), d.getMonth(), d.getDate())
  const diff = Math.floor((today.getTime() - day.getTime()) / 86400000)
  if (diff === 0) return 'Сегодня'
  if (diff === 1) return 'Вчера'
  return d.toLocaleDateString('ru-RU', { weekday: 'long', day: 'numeric', month: 'long' })
}
