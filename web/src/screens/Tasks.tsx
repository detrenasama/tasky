import { useEffect, useState } from 'react'
import { api } from '../api'
import type {
  Project,
  Task,
  Subtask,
  Link,
  Tag,
  TimeEntry,
  JournalEntry,
  ChecklistItem,
  StatusDef,
  StatusHistoryEntry,
  StatusOwner,
} from '../types'
import { fmtDuration, fmtDateTime } from '../fmt'
import { Button, useConfirm, MenuState, ContextMenu } from '../ui'

type Sel = { kind: 'task' | 'subtask'; id: number } | null

export default function Tasks({ onError }: { onError: (m: string) => void }) {
  const [projects, setProjects] = useState<Project[]>([])
  const [proj, setProj] = useState<Project | null>(null)
  const [tasks, setTasks] = useState<Task[]>([])
  const [subs, setSubs] = useState<Subtask[]>([])
  const [expanded, setExpanded] = useState<Set<number>>(new Set())
  const [sel, setSel] = useState<Sel>(null)
  const [statuses, setStatuses] = useState<StatusDef[]>([])
  const [runningId, setRunningId] = useState<number | null>(null)
  const [menu, setMenu] = useState<MenuState | null>(null)
  const confirm = useConfirm()[0]

  const [detail, setDetail] = useState<{
    description: string
    links: Link[]
    tags: Tag[]
    time: TimeEntry[]
    journal: JournalEntry[]
    checklist: ChecklistItem[]
    history: StatusHistoryEntry[]
  }>({ description: '', links: [], tags: [], time: [], journal: [], checklist: [], history: [] })

  useEffect(() => {
    api.statuses().then(setStatuses).catch((e) => onError(String(e)))
    api.projects().then((ps) => {
      setProjects(ps)
      if (ps[0]) openProject(ps[0])
    }).catch((e) => onError(String(e)))
    api.running().then((r) => setRunningId(r?.id ?? null)).catch(() => {})
  }, [])

  const openProject = (p: Project) => {
    setProj(p)
    setSel(null)
    Promise.all([api.tasksByProject(p.id), api.subtasksByProject(p.id)])
      .then(([ts, ss]) => {
        setTasks(ts)
        setSubs(ss)
      })
      .catch((e) => onError(String(e)))
  }

  const loadSubs = (taskId: number) => api.subtasksByTask(taskId)
  const toggle = (taskId: number) => {
    setExpanded((prev) => {
      const n = new Set(prev)
      if (n.has(taskId)) n.delete(taskId)
      else {
        n.add(taskId)
        loadSubs(taskId).then((ss) => setSubs((cur) => mergeSubs(cur, ss))).catch((e) => onError(String(e)))
      }
      return n
    })
  }

  const selectTask = (t: Task) => {
    setSel({ kind: 'task', id: t.id })
    api.taskDescription(t.id).then((d) => setDetail((x) => ({ ...x, description: d.description }))).catch((e) => onError(String(e)))
    api.taskLinks(t.id).then((l) => setDetail((x) => ({ ...x, links: l }))).catch((e) => onError(String(e)))
    api.taskTags(t.id).then((tg) => setDetail((x) => ({ ...x, tags: tg }))).catch((e) => onError(String(e)))
    setDetail((x) => ({ ...x, time: [], journal: [], checklist: [], history: [] }))
  }

  const selectSub = (s: Subtask) => {
    setSel({ kind: 'subtask', id: s.id })
    Promise.all([
      api.subtaskDescription(s.id),
      api.subtaskLinks(s.id),
      api.timeBySubtask(s.id),
      api.journal(s.id),
      api.checklist(s.id),
      api.statusHistory('subtask', s.id),
    ])
      .then(([d, l, t, j, c, h]) =>
        setDetail({ description: d.description, links: l, tags: [], time: t, journal: j, checklist: c, history: h }),
      )
      .catch((e) => onError(String(e)))
  }

  const owner: StatusOwner = sel?.kind === 'task' ? 'task' : 'subtask'

  const changeStatus = async (to: string, note = '') => {
    if (!sel) return
    try {
      await api.setStatus(owner, sel.id, to, note)
      if (proj) openProject(proj)
      if (sel.kind === 'subtask') selectSub({ ...({ id: sel.id } as Subtask) })
    } catch (e) {
      onError(String(e))
    }
  }

  const toggleTimer = async (s: Subtask) => {
    try {
      if (runningId === s.id) await api.stopSubtask(s.id)
      else await api.startSubtask(s.id)
      const r = await api.running()
      setRunningId(r?.id ?? null)
      if (proj) openProject(proj)
    } catch (e) {
      onError(String(e))
    }
  }

  const delTask = async (t: Task) => {
    if (!(await confirm(`Удалить задачу «${t.title}»?`))) return
    try {
      await api.deleteTask(t.id)
      if (proj) openProject(proj)
    } catch (e) {
      onError(String(e))
    }
  }
  const delSub = async (s: Subtask) => {
    if (!(await confirm(`Удалить подзадачу «${s.title}»?`))) return
    try {
      await api.deleteSubtask(s.id)
      if (proj) openProject(proj)
    } catch (e) {
      onError(String(e))
    }
  }

  const subsOf = (taskId: number) => subs.filter((s) => s.task_id === taskId)

  return (
    <div className="flex" style={{ alignItems: 'stretch', height: '100%' }}>
      <div className="panel col" style={{ width: 360, overflow: 'auto' }}>
        <select value={proj?.id ?? ''} onChange={(e) => {
          const p = projects.find((x) => x.id === Number(e.target.value))
          if (p) openProject(p)
        }}>
          {projects.map((p) => (
            <option key={p.id} value={p.id}>{p.name}</option>
          ))}
        </select>
        <ul className="list">
          {tasks.map((t) => (
            <li key={t.id} className="col" style={{ gap: 0 }}>
              <div
                className={`row ${sel?.kind === 'task' && sel.id === t.id ? 'selected' : ''}`}
                onClick={() => selectTask(t)}
                onContextMenu={(e) => {
                  e.preventDefault()
                  setMenu({ x: e.clientX, y: e.clientY, items: [{ label: 'Удалить', danger: true, onClick: () => delTask(t) }] })
                }}
              >
                <span className="title" style={{ cursor: 'pointer' }} onClick={(e) => { e.stopPropagation(); toggle(t.id) }}>
                  {expanded.has(t.id) ? '▾' : '▸'}
                </span>
                <span className="title">{t.title}</span>
                {t.sub_count > 0 && <span className="muted small">{t.sub_count}</span>}
                <Button className="danger small" onClick={(e) => { e.stopPropagation(); delTask(t) }}>✕</Button>
              </div>
              {expanded.has(t.id) && (
                <ul className="list" style={{ marginLeft: 16 }}>
                  {subsOf(t.id).map((s) => (
                    <li
                      key={s.id}
                      className={`row ${sel?.kind === 'subtask' && sel.id === s.id ? 'selected' : ''}`}
                      onClick={() => selectSub(s)}
                      onContextMenu={(e) => {
                        e.preventDefault()
                        setMenu({ x: e.clientX, y: e.clientY, items: [{ label: 'Удалить', danger: true, onClick: () => delSub(s) }] })
                      }}
                    >
                      <span className="status-dot" style={{ background: statusColor(s.status, statuses) }} />
                      <span className="title">{s.title}</span>
                      <span className="muted small">{fmtDuration(s.total_seconds)}</span>
                      {runningId === s.id && <span className="running">●</span>}
                      <Button className="danger small" onClick={(e) => { e.stopPropagation(); delSub(s) }}>✕</Button>
                    </li>
                  ))}
                </ul>
              )}
            </li>
          ))}
        </ul>
      </div>

      <div className="panel col" style={{ flex: 1, overflow: 'auto' }}>
        {!sel && <p className="muted">Выберите задачу или подзадачу.</p>}
        {sel?.kind === 'task' && <TaskDetail task={tasks.find((t) => t.id === sel.id)!} detail={detail} statuses={statuses} onError={onError} onStatus={changeStatus} onDesc={(v) => api.updateTaskDescription(sel.id, v).catch((e) => onError(String(e)))} onLinkAdd={async (n, u) => { await api.createTaskLink(sel.id, n, u); selectTask(tasks.find((t) => t.id === sel.id)!) }} onLinkDel={async (id) => { await api.deleteTaskLink(id); selectTask(tasks.find((t) => t.id === sel.id)!) }} onNewSub={async (title) => { await api.createSubtask(sel.id, title); if (proj) openProject(proj) }} onDel={() => delTask(tasks.find((t) => t.id === sel.id)!)}   onTagAdd={async (typeId, text, url) => { await api.createTag(sel.id, typeId, text, url); selectTask(tasks.find((t) => t.id === sel.id)!) }} onTagDel={async (id) => { await api.deleteTag(id); selectTask(tasks.find((t) => t.id === sel.id)!) }} />}
        {sel?.kind === 'subtask' && <SubDetail sub={subs.find((s) => s.id === sel.id)!} detail={detail} statuses={statuses} running={runningId === sel.id} onError={onError} onStatus={changeStatus} onToggleTimer={() => toggleTimer(subs.find((s) => s.id === sel.id)!)} onDesc={(v) => api.updateSubtaskDescription(sel.id, v).catch((e) => onError(String(e)))} onLinkAdd={async (n, u) => { await api.createSubtaskLink(sel.id, n, u); selectSub({ ...({ id: sel.id } as Subtask) }) }} onLinkDel={async (id) => { await api.deleteSubtaskLink(id); selectSub({ ...({ id: sel.id } as Subtask) }) }} onTimeEdit={async (id, s, e) => { await api.updateTimeEntry(id, s, e); selectSub({ ...({ id: sel.id } as Subtask) }) }} onTimeDel={async (id) => { await api.deleteTimeEntry(id); selectSub({ ...({ id: sel.id } as Subtask) }) }} onJournalAdd={async (text) => { await api.createJournal(sel.id, text); selectSub({ ...({ id: sel.id } as Subtask) }) }} onCheckToggle={async (id, st) => { await api.setChecklistStatus(id, st); selectSub({ ...({ id: sel.id } as Subtask) }) }} onCheckAdd={async (text) => { await api.createChecklistItem(sel.id, text); selectSub({ ...({ id: sel.id } as Subtask) }) }} onCheckDel={async (id) => { await api.deleteChecklistItem(id); selectSub({ ...({ id: sel.id } as Subtask) }) }} />}
      </div>

      <ContextMenu state={menu} onClose={() => setMenu(null)} />
    </div>
  )
}

function statusColor(name: string, statuses: StatusDef[]): string {
  const s = statuses.find((x) => x.name === name)
  return s?.color || 'var(--grey)'
}

function mergeSubs(cur: Subtask[], add: Subtask[]): Subtask[] {
  const byId = new Map(cur.map((s) => [s.id, s]))
  add.forEach((s) => byId.set(s.id, s))
  return Array.from(byId.values())
}

// --- Детальные панели ---

function TaskDetail(props: {
  task: Task
  detail: any
  statuses: StatusDef[]
  onError: (m: string) => void
  onStatus: (to: string, note?: string) => void
  onDesc: (v: string) => void
  onLinkAdd: (n: string, u: string) => void
  onLinkDel: (id: number) => void
  onNewSub: (title: string) => void
  onDel: () => void
  onTagAdd: (typeId: number, text: string, url: string) => void
  onTagDel: (id: number) => void
}) {
  const [desc, setDesc] = useState(props.detail.description)
  useEffect(() => setDesc(props.detail.description), [props.detail.description])
  const [st, setSt] = useState('')
  const [nt, setNt] = useState('')
  const [subTitle, setSubTitle] = useState('')

  return (
    <div className="col">
      <div className="flex between">
        <h2 style={{ margin: 0 }}>{props.task.title}</h2>
        <Button className="danger" onClick={props.onDel}>Удалить</Button>
      </div>
      <div>
        <div className="flex between"><strong>Описание</strong><Button onClick={() => props.onDesc(desc)}>Сохранить</Button></div>
        <textarea value={desc} onChange={(e) => setDesc(e.target.value)} />
      </div>
      <div className="flex" style={{ gap: 8 }}>
        <select value={st} onChange={(e) => setSt(e.target.value)}>
          <option value="">Сменить статус…</option>
          {props.statuses.map((s) => (<option key={s.id} value={s.name}>{s.name}</option>))}
        </select>
        {st && <input placeholder="Заметка" value={nt} onChange={(e) => setNt(e.target.value)} />}
        <Button className="primary" disabled={!st} onClick={() => { props.onStatus(st, nt); setSt(''); setNt('') }}>Применить</Button>
      </div>
      <div>
        <div className="flex between"><strong>Подзадачи</strong></div>
        <div className="flex">
          <input placeholder="Название подзадачи" value={subTitle} onChange={(e) => setSubTitle(e.target.value)} />
          <Button className="primary" disabled={!subTitle.trim()} onClick={() => { props.onNewSub(subTitle.trim()); setSubTitle('') }}>＋</Button>
        </div>
      </div>
      <LinksBlock links={props.detail.links} onAdd={props.onLinkAdd} onDel={props.onLinkDel} />
      <TagsBlock tags={props.detail.tags} onAdd={props.onTagAdd} onDel={props.onTagDel} />
    </div>
  )
}

function SubDetail(props: {
  sub: Subtask
  detail: any
  statuses: StatusDef[]
  running: boolean
  onError: (m: string) => void
  onStatus: (to: string, note?: string) => void
  onToggleTimer: () => void
  onDesc: (v: string) => void
  onLinkAdd: (n: string, u: string) => void
  onLinkDel: (id: number) => void
  onTimeEdit: (id: number, started: string, ended: string | null) => void
  onTimeDel: (id: number) => void
  onJournalAdd: (text: string) => void
  onCheckToggle: (id: number, status: string) => void
  onCheckAdd: (text: string) => void
  onCheckDel: (id: number) => void
}) {
  const [desc, setDesc] = useState(props.detail.description)
  useEffect(() => setDesc(props.detail.description), [props.detail.description])
  const [st, setSt] = useState('')
  const [nt, setNt] = useState('')
  const [jt, setJt] = useState('')
  const [ct, setCt] = useState('')

  const checkStates = ['new', 'in_progress', 'done', 'cancelled']

  return (
    <div className="col">
      <div className="flex between">
        <h2 style={{ margin: 0 }}>{props.sub.title}</h2>
        <Button className={props.running ? 'danger' : 'primary'} onClick={props.onToggleTimer}>
          {props.running ? 'Стоп' : 'Старт'}
        </Button>
      </div>
      <div className="flex" style={{ gap: 8 }}>
        <select value={st} onChange={(e) => setSt(e.target.value)}>
          <option value="">Сменить статус…</option>
          {props.statuses.map((s) => (<option key={s.id} value={s.name}>{s.name}</option>))}
        </select>
        {st && <input placeholder="Заметка" value={nt} onChange={(e) => setNt(e.target.value)} />}
        <Button className="primary" disabled={!st} onClick={() => { props.onStatus(st, nt); setSt(''); setNt('') }}>Применить</Button>
      </div>
      <div>
        <div className="flex between"><strong>Описание</strong><Button onClick={() => props.onDesc(desc)}>Сохранить</Button></div>
        <textarea value={desc} onChange={(e) => setDesc(e.target.value)} />
      </div>

      <div>
        <strong>Записи времени</strong>
        <ul className="list">
          {props.detail.time.map((t: TimeEntry) => (
            <li key={t.id} className="row">
              <span className="title">{fmtDateTime(t.started_at)} — {t.ended_at ? fmtDateTime(t.ended_at) : '…'}</span>
              <Button onClick={() => {
                const s = prompt('Начало (ISO)', t.started_at)
                if (!s) return
                const e = prompt('Конец (ISO, пусто = открыто)', t.ended_at ?? '')
                props.onTimeEdit(t.id, s, e || null)
              }}>✎</Button>
              <Button className="danger" onClick={() => props.onTimeDel(t.id)}>✕</Button>
            </li>
          ))}
        </ul>
      </div>

      <div>
        <strong>Журнал</strong>
        <div className="flex">
          <input placeholder="Новая запись" value={jt} onChange={(e) => setJt(e.target.value)} />
          <Button className="primary" disabled={!jt.trim()} onClick={() => { props.onJournalAdd(jt.trim()); setJt('') }}>＋</Button>
        </div>
        <ul className="list">
          {props.detail.journal.map((j: JournalEntry) => (
            <li key={j.id} className="row"><span className="title">{j.text}</span><span className="muted small">{fmtDateTime(j.created_at)}</span></li>
          ))}
        </ul>
      </div>

      <div>
        <strong>Чек-лист</strong>
        <div className="flex">
          <input placeholder="Пункт" value={ct} onChange={(e) => setCt(e.target.value)} />
          <Button className="primary" disabled={!ct.trim()} onClick={() => { props.onCheckAdd(ct.trim()); setCt('') }}>＋</Button>
        </div>
        <ul className="list">
          {props.detail.checklist.map((c: ChecklistItem) => (
            <li key={c.id} className="row">
              <select value={c.status} onChange={(e) => props.onCheckToggle(c.id, e.target.value)}>
                {checkStates.map((s) => (<option key={s} value={s}>{s}</option>))}
              </select>
              <span className="title">{c.text}</span>
              <Button className="danger" onClick={() => props.onCheckDel(c.id)}>✕</Button>
            </li>
          ))}
        </ul>
      </div>

      <LinksBlock links={props.detail.links} onAdd={props.onLinkAdd} onDel={props.onLinkDel} />
    </div>
  )
}

function LinksBlock({ links, onAdd, onDel }: { links: Link[]; onAdd: (n: string, u: string) => void; onDel: (id: number) => void }) {
  const [ln, setLn] = useState({ name: '', url: '' })
  return (
    <div>
      <div className="flex between"><strong>Ссылки</strong></div>
      <div className="flex">
        <input placeholder="Название" value={ln.name} onChange={(e) => setLn({ ...ln, name: e.target.value })} />
        <input placeholder="URL" value={ln.url} onChange={(e) => setLn({ ...ln, url: e.target.value })} />
        <Button className="primary" disabled={!ln.url.trim()} onClick={() => { onAdd(ln.name, ln.url); setLn({ name: '', url: '' }) }}>＋</Button>
      </div>
      <ul className="list">
        {links.map((l) => (
          <li key={l.id} className="row">
            <a href={l.url} target="_blank" rel="noreferrer">{l.name || l.url}</a>
            <Button className="danger" onClick={() => onDel(l.id)}>✕</Button>
          </li>
        ))}
      </ul>
    </div>
  )
}

function TagsBlock({ tags, onAdd, onDel }: { tags: Tag[]; onAdd: (typeId: number, text: string, url: string) => void; onDel: (id: number) => void }) {
  const [t, setT] = useState({ typeId: 0, text: '', url: '' })
  return (
    <div>
      <div className="flex between"><strong>Теги</strong></div>
      <div className="flex">
        <input type="number" placeholder="type_id" value={t.typeId || ''} onChange={(e) => setT({ ...t, typeId: Number(e.target.value) })} />
        <input placeholder="Текст" value={t.text} onChange={(e) => setT({ ...t, text: e.target.value })} />
        <input placeholder="URL" value={t.url} onChange={(e) => setT({ ...t, url: e.target.value })} />
        <Button className="primary" disabled={!t.typeId || !t.text.trim()} onClick={() => { onAdd(t.typeId, t.text, t.url); setT({ typeId: 0, text: '', url: '' }) }}>＋</Button>
      </div>
      <div>
        {tags.map((tg) => (
          <span key={tg.id} className="tag" style={{ background: tg.color || 'var(--panel2)' }}>
            {tg.type_name}: {tg.text}
            <span style={{ cursor: 'pointer' }} onClick={() => onDel(tg.id)}> ✕</span>
          </span>
        ))}
      </div>
    </div>
  )
}
