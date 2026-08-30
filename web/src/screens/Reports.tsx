import { useEffect, useState } from 'react'
import { api } from '../api'
import type { ReportEntry, ReportJournalEntry, Tag, Project } from '../types'
import { fmtDuration, fmtDateTime } from '../fmt'
import { Button } from '../ui'

type Period = 'today' | 'yesterday' | 'week' | 'month' | 'custom'

function rangeOf(p: Period, from?: string, to?: string): [string, string] {
  const now = new Date()
  const startOfDay = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate())
  switch (p) {
    case 'today': {
      const a = startOfDay(now)
      return [a.toISOString(), new Date(a.getTime() + 86400000).toISOString()]
    }
    case 'yesterday': {
      const a = startOfDay(new Date(now.getTime() - 86400000))
      return [a.toISOString(), new Date(a.getTime() + 86400000).toISOString()]
    }
    case 'week': {
      const dow = (now.getDay() + 6) % 7
      const a = startOfDay(new Date(now.getTime() - dow * 86400000))
      return [a.toISOString(), new Date(a.getTime() + 7 * 86400000).toISOString()]
    }
    case 'month': {
      const a = new Date(now.getFullYear(), now.getMonth(), 1)
      return [a.toISOString(), new Date(now.getFullYear(), now.getMonth() + 1, 1).toISOString()]
    }
    default:
      return [new Date(from + 'T00:00:00').toISOString(), new Date(to + 'T00:00:00').toISOString()]
  }
}

export default function Reports({ onError }: { onError: (m: string) => void }) {
  const [period, setPeriod] = useState<Period>('today')
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [projectId, setProjectId] = useState(0)
  const [projects, setProjects] = useState<Project[]>([])
  const [entries, setEntries] = useState<ReportEntry[]>([])
  const [journal, setJournal] = useState<ReportJournalEntry[]>([])
  const [tags, setTags] = useState<Record<string, Tag[]>>({})
  const [withJournal, setWithJournal] = useState(false)
  const [saved, setSaved] = useState('')

  const run = () => {
    const [f, t] = rangeOf(period, from, to)
    api
      .reports(f, t, projectId)
      .then(async (es) => {
        setEntries(es)
        if (withJournal) {
          const j = await api.reportsJournal(f, t).catch(() => [])
          setJournal(j)
        } else setJournal([])
        const ids = Array.from(new Set(es.map((e) => e.task_id)))
        if (ids.length) {
          const tg = await api.reportsTags(ids).catch(() => ({} as Record<string, Tag[]>))
          setTags(tg)
        } else setTags({})
      })
      .catch((e) => onError(String(e)))
  }

  useEffect(() => {
    api.projects().then(setProjects).catch((e) => onError(String(e)))
  }, [])
  useEffect(run, [])

  const save = () => {
    const [f, t] = rangeOf(period, from, to)
    let txt = `Отчёт ${f.slice(0, 10)} — ${t.slice(0, 10)}\n\n`
    const byTask = new Map<string, ReportEntry[]>()
    entries.forEach((e) => {
      const k = e.task_title
      if (!byTask.has(k)) byTask.set(k, [])
      byTask.get(k)!.push(e)
    })
    byTask.forEach((subs, task) => {
      const total = subs.reduce((a, b) => a + b.seconds, 0)
      txt += `${task} — ${fmtDuration(total)}\n`
      const tgs = tags[String(subs[0].task_id)] || []
      if (tgs.length) txt += `  теги: ${tgs.map((x) => x.text).join(', ')}\n`
      subs.forEach((s) => {
        txt += `  ├ ${s.subtask_title} — ${fmtDuration(s.seconds)}\n`
      })
      txt += '\n'
    })
    const blob = new Blob([txt], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `tasky-report-${f.slice(0, 10)}.txt`
    a.click()
    URL.revokeObjectURL(url)
    setSaved(a.download)
  }

  return (
    <div className="panel col" style={{ flex: 1, overflow: 'auto' }}>
      <div className="toolbar">
        {(['today', 'yesterday', 'week', 'month', 'custom'] as Period[]).map((p) => (
          <Button key={p} color={period === p ? 'accent' : 'base'} onClick={() => setPeriod(p)}>
            {p === 'today' ? 'Сегодня' : p === 'yesterday' ? 'Вчера' : p === 'week' ? 'Неделя' : p === 'month' ? 'Месяц' : 'Свой'}
          </Button>
        ))}
        {period === 'custom' && (
          <span className="flex">
            <input type="date" value={from} onChange={(e) => setFrom(e.target.value)} />
            <input type="date" value={to} onChange={(e) => setTo(e.target.value)} />
          </span>
        )}
        <label className="flex"><input type="checkbox" checked={withJournal} onChange={(e) => setWithJournal(e.target.checked)} /> Журнал</label>
        <select value={projectId} onChange={(e) => setProjectId(Number(e.target.value))}>
          <option value={0}>Все проекты</option>
          {projects.map((p) => (<option key={p.id} value={p.id}>{p.name}</option>))}
        </select>
        <Button icon="refresh" onClick={run}>Обновить</Button>
        <Button color="accent" icon="save" onClick={save}>Сохранить</Button>
      </div>

      {entries.length === 0 && <p className="muted">Времени за период ещё не учтено.</p>}

      {(() => {
        const byTask = new Map<string, ReportEntry[]>()
        entries.forEach((e) => {
          if (!byTask.has(e.task_title)) byTask.set(e.task_title, [])
          byTask.get(e.task_title)!.push(e)
        })
        const journalsBySub = new Map<number, ReportJournalEntry[]>()
        journal.forEach((j) => {
          if (!journalsBySub.has(j.subtask_id)) journalsBySub.set(j.subtask_id, [])
          journalsBySub.get(j.subtask_id)!.push(j)
        })
        return Array.from(byTask.entries()).map(([task, subs]) => {
          const total = subs.reduce((a, b) => a + b.seconds, 0)
          const tgs = tags[String(subs[0].task_id)] || []
          return (
            <div key={task} className="panel" style={{ marginBottom: 8 }}>
              <strong>{task} — {fmtDuration(total)}</strong>
              {tgs.length > 0 && <span className="muted small">  [{tgs.map((x) => x.text).join(', ')}]</span>}
              <ul className="list">
                {subs.map((s) => (
                  <li key={s.subtask_id} className="row">
                    <span className="title" style={{ paddingLeft: 12 }}>├ {s.subtask_title} — {fmtDuration(s.seconds)}</span>
                    {withJournal && (journalsBySub.get(s.subtask_id) || []).map((j) => (
                      <div key={j.subtask_id + j.created_at} className="muted small" style={{ paddingLeft: 24 }}>
                        [{fmtDateTime(j.created_at)}] {j.text}
                      </div>
                    ))}
                  </li>
                ))}
              </ul>
            </div>
          )
        })
      })()}

      {saved && <p className="muted small">Сохранён отчёт: {saved}</p>}
    </div>
  )
}
