import { useEffect, useState } from 'react'
import { api } from '../api'
import type { ReportEntry, ReportJournalEntry, Tag, Project, ChecklistItem } from '../types'
import { fmtDuration, fmtDateTime } from '../fmt'
import { Button } from '../ui'
import { TagBar } from '../ui/tagBar'

type Period = 'today' | 'yesterday' | 'week' | 'month' | 'custom'

const monthNames = ['январь','февраль','март','апрель','май','июнь','июль','август','сентябрь','октябрь','ноябрь','декабрь']
const pad2 = (n: number) => String(n).padStart(2,'0')

function fmtDDMMYYYY(d: Date): string { return `${pad2(d.getDate())}.${pad2(d.getMonth()+1)}.${d.getFullYear()}` }
function fmtDDMM(d: Date): string { return `${pad2(d.getDate())}.${pad2(d.getMonth()+1)}` }

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
    default: {
      if (!from || !to) return ['', '']
      const a = new Date(from + 'T00:00:00')
      const b = new Date(to + 'T00:00:00')
      if (isNaN(a.getTime()) || isNaN(b.getTime())) return ['', '']
      return [a.toISOString(), b.toISOString()]
    }
  }
}

function periodLabel(p: Period, fISO: string, tISO: string): string {
  if (p === 'custom') {
    if (!fISO || !tISO) return 'Отчет · свой период'
    const f = new Date(fISO)
    const t = new Date(tISO)
    if (isNaN(f.getTime()) || isNaN(t.getTime())) return 'Отчет · свой период'
    const diff = t.getTime() - f.getTime()
    const end = new Date(t.getTime() - 86400000)
    if (diff <= 24*3600000) return `Отчет за день · ${fmtDDMMYYYY(f)}`
    return `Отчет · ${fmtDDMMYYYY(f)} – ${fmtDDMMYYYY(end)}`
  }
  const f = fISO ? new Date(fISO) : new Date()
  if (isNaN(f.getTime())) return 'Отчет за сегодня · ' + fmtDDMMYYYY(new Date())
  switch (p) {
    case 'yesterday': return `Отчет за вчера · ${fmtDDMMYYYY(f)}`
    case 'week': {
      const t = tISO ? new Date(tISO) : new Date()
      const end = new Date(t.getTime() - 86400000)
      return `Отчет за неделю · ${fmtDDMM(f)} – ${fmtDDMMYYYY(end)}`
    }
    case 'month': return `Отчет за ${monthNames[f.getMonth()]} ${f.getFullYear()}`
    default: return `Отчет за сегодня · ${fmtDDMMYYYY(f)}`
  }
}

// TUI-правило reports_screen.go:checklistSection — done только если статус сменился в [from,to), in_progress всегда, new/cancelled скрыты.
function checklistSections(items: ChecklistItem[], from: string, to: string): { done: ChecklistItem[]; inProgress: ChecklistItem[] } {
  if (!items.length || !from || !to) {
    // in_progress показываются даже без периода, done — только с валидным периодом
    const inProgress = items.filter((it) => it.status === 'in_progress')
    if (!from || !to) return { done: [], inProgress }
    const fromD = new Date(from)
    const toD = new Date(to)
    if (isNaN(fromD.getTime()) || isNaN(toD.getTime())) return { done: [], inProgress }
    const done = items.filter((it) => {
      if (it.status !== 'done') return false
      const d = new Date(it.status_changed_at)
      if (isNaN(d.getTime())) return false
      return d >= fromD && d < toD
    })
    return { done, inProgress }
  }
  const fromD = new Date(from)
  const toD = new Date(to)
  const done: ChecklistItem[] = []
  const inProgress: ChecklistItem[] = []
  for (const it of items) {
    if (it.status === 'in_progress') inProgress.push(it)
    else if (it.status === 'done') {
      if (isNaN(fromD.getTime()) || isNaN(toD.getTime())) continue
      const d = new Date(it.status_changed_at)
      if (isNaN(d.getTime())) continue
      if (d >= fromD && d < toD) done.push(it)
    }
  }
  return { done, inProgress }
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
  const [checklistMap, setChecklistMap] = useState<Record<string, ChecklistItem[]>>({})
  const [withJournal, setWithJournal] = useState(false)
  const [saved, setSaved] = useState('')

  useEffect(() => {
    api.projects().then(setProjects).catch((e) => onError(String(e)))
  }, [])

  useEffect(() => {
    if (period === 'custom' && (!from || !to)) {
      setEntries([])
      setJournal([])
      setTags({})
      setChecklistMap({})
      return
    }
    const [f, t] = rangeOf(period, from, to)
    if (!f || !t) {
      setEntries([])
      setJournal([])
      setTags({})
      setChecklistMap({})
      return
    }
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
        // чек-лист по каждой подзадаче отчёта (правила как в TUI)
        const subIds = Array.from(new Set(es.map((e) => e.subtask_id)))
        if (subIds.length) {
          const results = await Promise.all(subIds.map((id) => api.checklist(id).catch(() => [] as ChecklistItem[])))
          const m: Record<string, ChecklistItem[]> = {}
          subIds.forEach((id, i) => { m[String(id)] = results[i] })
          setChecklistMap(m)
        } else setChecklistMap({})
      })
      .catch((e) => onError(String(e)))
  }, [period, from, to, projectId, withJournal])

  const save = () => {
    const [f, t] = rangeOf(period, from, to)
    const label = periodLabel(period, f, t)
    const overall = entries.reduce((a,b)=>a+b.seconds,0)
    let txt = `${label}\n\n`
    // группировка по проекту → задаче (как в рендере)
    const byProject = new Map<number, { name: string; tasks: Map<number, { title: string; subs: ReportEntry[] }> }>()
    entries.forEach((e) => {
      if (!byProject.has(e.project_id)) byProject.set(e.project_id, { name: e.project_name, tasks: new Map() })
      const g = byProject.get(e.project_id)!
      if (!g.tasks.has(e.task_id)) g.tasks.set(e.task_id, { title: e.task_title, subs: [] })
      g.tasks.get(e.task_id)!.subs.push(e)
    })
    for (const [, proj] of byProject) {
      const projectTotal = Array.from(proj.tasks.values()).flatMap(g=>g.subs).reduce((a,b)=>a+b.seconds,0)
      txt += `${proj.name} · ${fmtDuration(projectTotal)}\n`
      for (const [, grp] of proj.tasks) {
        const total = grp.subs.reduce((a, b) => a + b.seconds, 0)
        txt += `${grp.title} — ${fmtDuration(total)}\n`
        const tgs = tags[String(grp.subs[0].task_id)] || []
        if (tgs.length) txt += `  теги: ${tgs.map((x) => x.text).join(', ')}\n`
        for (const s of grp.subs) {
          txt += `  ├ ${s.subtask_title} — ${fmtDuration(s.seconds)}\n`
          if (withJournal) {
            const js = journal.filter((j) => j.subtask_id === s.subtask_id)
            for (const j of js) txt += `      [${j.created_at.slice(0, 16)}] ${j.text}\n`
          }
          const ch = checklistMap[String(s.subtask_id)] || []
          const { done, inProgress } = checklistSections(ch, f, t)
          if (done.length) {
            txt += `    Выполнены:\n`
            for (const it of done) txt += `      • ${it.text}\n`
          }
          if (inProgress.length) {
            txt += `    В работе:\n`
            for (const it of inProgress) txt += `      • ${it.text}\n`
          }
        }
        txt += '\n'
      }
    }
    txt += `Общее время: ${fmtDuration(overall)}\n`
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
        <Button color="accent" icon="save" onClick={save}>Сохранить</Button>
      </div>

      {(() => {
        const [fISO, tISO] = rangeOf(period, from, to)
        const overall = entries.reduce((a,b)=>a+b.seconds,0)
        const label = periodLabel(period, fISO, tISO)
        return (
          <div className="panel" style={{ padding:'6px 10px', marginBottom:8 }}>
            <span>{label}  <span className="muted">Общее время: {fmtDuration(overall)}</span></span>
          </div>
        )
      })()}

      {entries.length === 0 && <p className="muted">Времени за период ещё не учтено.</p>}

      {(() => {
        const journalsBySub = new Map<number, ReportJournalEntry[]>()
        journal.forEach((j) => {
          if (!journalsBySub.has(j.subtask_id)) journalsBySub.set(j.subtask_id, [])
          journalsBySub.get(j.subtask_id)!.push(j)
        })
        const byProject = new Map<number, { name: string; tasks: Map<number, { title: string; subs: ReportEntry[] }> }>()
        entries.forEach((e) => {
          if (!byProject.has(e.project_id)) byProject.set(e.project_id, { name: e.project_name, tasks: new Map() })
          const g = byProject.get(e.project_id)!
          if (!g.tasks.has(e.task_id)) g.tasks.set(e.task_id, { title: e.task_title, subs: [] })
          g.tasks.get(e.task_id)!.subs.push(e)
        })
        const [f, t] = rangeOf(period, from, to)
        if (byProject.size === 0) return null
        const projectsView = Array.from(byProject.entries()).map(([pid, proj]) => {
          const projectTotal = Array.from(proj.tasks.values()).flatMap(g=>g.subs).reduce((a,b)=>a+b.seconds,0)
          return (
          <div key={pid} style={{ marginBottom: 12 }}>
            <div style={{ fontWeight: 600, margin: '8px 0 6px', fontSize: 14 }}>{proj.name} · {fmtDuration(projectTotal)}</div>
            {Array.from(proj.tasks.entries()).map(([tid, grp]) => {
              const total = grp.subs.reduce((a, b) => a + b.seconds, 0)
              const tgs = tags[String(tid)] || []
              return (
                <div key={tid} className="panel" style={{ marginBottom: 8 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
                    <strong>{grp.title} — {fmtDuration(total)}</strong>
                    {tgs.length > 0 && <TagBar tags={tgs} tagTypes={[]} readOnly />}
                  </div>
                  <ul className="list" style={{ marginTop: 6 }}>
                    {grp.subs.map((s) => {
                      const ch = checklistMap[String(s.subtask_id)] || []
                      const { done, inProgress } = checklistSections(ch, f, t)
                      return (
                        <li key={s.subtask_id} className="col" style={{ alignItems: 'stretch', gap: 4, paddingLeft: 12 }}>
                          <span className="title">├ {s.subtask_title} — {fmtDuration(s.seconds)}</span>
                          {withJournal && (journalsBySub.get(s.subtask_id) || []).map((j) => (
                            <div key={j.subtask_id + j.created_at} className="muted small" style={{ paddingLeft: 12 }}>
                              [{fmtDateTime(j.created_at)}] {j.text}
                            </div>
                          ))}
                          {(done.length > 0 || inProgress.length > 0) && (
                            <div style={{ paddingLeft: 12, fontSize: 13, display: 'flex', flexDirection: 'column', gap: 2 }}>
                              {done.length > 0 && (
                                <>
                                  <span className="muted small">Выполнены:</span>
                                  {done.map((it) => <span key={it.id}>• {it.text}</span>)}
                                </>
                              )}
                              {inProgress.length > 0 && (
                                <>
                                  <span className="muted small">В работе:</span>
                                  {inProgress.map((it) => <span key={it.id}>• {it.text}</span>)}
                                </>
                              )}
                            </div>
                          )}
                        </li>
                      )
                    })}
                  </ul>
                </div>
              )
            })}
          </div>
        )
        })
        return (<>{projectsView}</>)
      })()}

      {saved && <p className="muted small">Сохранён отчёт: {saved}</p>}
    </div>
  )
}
