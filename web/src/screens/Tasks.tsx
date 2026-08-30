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
} from '../types'
import { fmtDuration, fmtDateTime } from '../fmt'
import { Button, useConfirm, useColumnWidth, Modal } from '../ui'

type Sel = { kind: 'task' | 'subtask'; id: number } | null

type Detail = {
  description: string
  links: Link[]
  tags: Tag[]
  time: TimeEntry[]
  journal: JournalEntry[]
  checklist: ChecklistItem[]
  history: StatusHistoryEntry[]
}

const EMPTY: Detail = { description: '', links: [], tags: [], time: [], journal: [], checklist: [], history: [] }

export default function Tasks({ onError }: { onError: (m: string) => void }) {
  const [projects, setProjects] = useState<Project[]>([])
  const [proj, setProj] = useState<Project | null>(null)
  const [tasks, setTasks] = useState<Task[]>([])
  const [subs, setSubs] = useState<Subtask[]>([])
  const [selTaskId, setSelTaskId] = useState<number | null>(null)
  const [selSubId, setSelSubId] = useState<number | null>(null)
  const [taskDetail, setTaskDetail] = useState<Detail>(EMPTY)
  const [subDetail, setSubDetail] = useState<Detail>(EMPTY)
  const [statuses, setStatuses] = useState<StatusDef[]>([])
  const [runningId, setRunningId] = useState<number | null>(null)
  const [tagsMap, setTagsMap] = useState<Record<string, Tag[]>>({})
  const [checkMap, setCheckMap] = useState<Record<string, [number, number]>>({})

  const [addKind, setAddKind] = useState<null | 'task' | 'sub'>(null)
  const [addTitle, setAddTitle] = useState('')
  const [confirm, confirmNode] = useConfirm()

  const taskResize = useColumnWidth(320, 'tasky.col.tasks')
  const subResize = useColumnWidth(300, 'tasky.col.subs')

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
    setSelTaskId(null)
    setSelSubId(null)
    setTaskDetail(EMPTY)
    setSubDetail(EMPTY)
    Promise.all([api.tasksByProject(p.id), api.subtasksByProject(p.id)])
      .then(([ts, ss]) => {
        setTasks(ts)
        setSubs(ss)
      })
      .catch((e) => onError(String(e)))
    api.tagsByProject(p.id).then(setTagsMap).catch(() => setTagsMap({}))
    api.checklistCounts(p.id).then(setCheckMap).catch(() => setCheckMap({}))
  }

  // reload — перечитывает задачи/подзадачи проекта без сброса выбора.
  const reload = () => {
    if (!proj) return Promise.resolve()
    const pid = proj.id
    api.tagsByProject(pid).then(setTagsMap).catch(() => {})
    api.checklistCounts(pid).then(setCheckMap).catch(() => {})
    return Promise.all([api.tasksByProject(pid), api.subtasksByProject(pid)])
      .then(([ts, ss]) => {
        setTasks(ts)
        setSubs(ss)
      })
      .catch((e) => onError(String(e)))
  }

  const loadTaskDetail = (id: number) => {
    Promise.all([api.taskDescription(id), api.taskLinks(id), api.taskTags(id)])
      .then(([d, l, tg]) => setTaskDetail({ description: d.description, links: l, tags: tg, time: [], journal: [], checklist: [], history: [] }))
      .catch((e) => onError(String(e)))
  }
  const loadSubDetail = (id: number) => {
    Promise.all([
      api.subtaskDescription(id),
      api.subtaskLinks(id),
      api.timeBySubtask(id),
      api.journal(id),
      api.checklist(id),
      api.statusHistory('subtask', id),
    ])
      .then(([d, l, t, j, c, h]) => setSubDetail({ description: d.description, links: l, tags: [], time: t, journal: j, checklist: c, history: h }))
      .catch((e) => onError(String(e)))
  }

  const selectTask = (t: Task) => {
    setSelTaskId(t.id)
    setSelSubId(null)
    setSubDetail(EMPTY)
    loadTaskDetail(t.id)
  }

  const selectSub = (s: Subtask) => {
    setSelSubId(s.id)
    if (selTaskId == null) setSelTaskId(s.task_id)
    loadSubDetail(s.id)
  }

  // Создание задачи/подзадачи через модалку. Новый элемент сразу выбирается.
  const openAdd = (kind: 'task' | 'sub') => {
    setAddTitle('')
    setAddKind(kind)
  }
  const submitAdd = async () => {
    const title = addTitle.trim()
    if (!title || !addKind) return
    try {
      if (addKind === 'task') {
        if (!proj) return
        const t = await api.createTask(proj.id, title)
        await reload()
        setSelTaskId(t.id)
        setSelSubId(null)
        setSubDetail(EMPTY)
        loadTaskDetail(t.id)
      } else {
        if (!selTaskId) return
        const s = await api.createSubtask(selTaskId, title)
        await reload()
        setSelSubId(s.id)
        if (selTaskId == null) setSelTaskId(s.task_id)
        loadSubDetail(s.id)
      }
    } catch (e) {
      onError(String(e))
    }
    setAddKind(null)
  }

  const active: Sel = selSubId != null
    ? { kind: 'subtask', id: selSubId }
    : selTaskId != null
      ? { kind: 'task', id: selTaskId }
      : null

  const changeStatus = async (to: string, note = '') => {
    if (!active) return
    try {
      await api.setStatus(active.kind, active.id, to, note)
      reload()
      if (active.kind === 'subtask') loadSubDetail(active.id)
      else loadTaskDetail(active.id)
    } catch (e) {
      onError(String(e))
    }
  }

  const toggleTimer = async () => {
    if (selSubId == null) return
    const s = subs.find((x) => x.id === selSubId)
    if (!s) return
    try {
      if (runningId === s.id) await api.stopSubtask(s.id)
      else await api.startSubtask(s.id)
      const r = await api.running()
      setRunningId(r?.id ?? null)
      reload()
    } catch (e) {
      onError(String(e))
    }
  }

  const delTask = async (t: Task) => {
    if (!(await confirm(`Удалить задачу «${t.title}»?`))) return
    try {
      await api.deleteTask(t.id)
      if (selTaskId === t.id) {
        setSelTaskId(null)
        setSelSubId(null)
        setTaskDetail(EMPTY)
        setSubDetail(EMPTY)
      }
      reload()
    } catch (e) {
      onError(String(e))
    }
  }
  const delSub = async (s: Subtask) => {
    if (!(await confirm(`Удалить подзадачу «${s.title}»?`))) return
    try {
      await api.deleteSubtask(s.id)
      if (selSubId === s.id) {
        setSelSubId(null)
        setSubDetail(EMPTY)
      }
      reload()
    } catch (e) {
      onError(String(e))
    }
  }

  const subsOf = (taskId: number) => subs.filter((s) => s.task_id === taskId)

  // Хелперы для двухстрочных карточек
  const doneSet = new Set(statuses.filter((s) => s.type === 'done').map((s) => s.name))
  const isDone = (name: string) => doneSet.has(name)
  const taskTime = (taskId: number) =>
    subs
      .filter((s) => s.task_id === taskId)
      .reduce((a, s) => {
        let t = s.total_seconds
        if (s.active_since) t += Math.floor(Date.now() / 1000) - s.active_since
        return a + t
      }, 0)
  const taskDoneCount = (taskId: number) =>
    subs.filter((s) => s.task_id === taskId && doneSet.has(s.status)).length

  return (
    <div className="flex" style={{ alignItems: 'stretch', height: '100%' }}>
      {/* Колонка 1: проект + задачи */}
      <div className="panel col" style={{ width: taskResize.width, position: 'relative', overflow: 'auto' }}>
        <select value={proj?.id ?? ''} onChange={(e) => {
          const p = projects.find((x) => x.id === Number(e.target.value))
          if (p) openProject(p)
        }}>
          {projects.map((p) => (
            <option key={p.id} value={p.id}>{p.name}</option>
          ))}
        </select>
        <div className="toolbar">
          {proj && <Button color="accent" icon="plus" label="Создать задачу" onClick={() => openAdd('task')} />}
        </div>
        {proj ? (
          <ul className="list">
            {tasks.map((t) => {
              const col = statusColor(t.status, statuses)
              const done = isDone(t.status)
              const secs = taskTime(t.id)
              const hasTime = secs > 0
              const hasCount = t.sub_count > 0
              const k = taskDoneCount(t.id)
              const tags = tagsMap[String(t.id)] || []
              return (
                <li
                  key={t.id}
                  className={`row task-row${selTaskId === t.id ? ' selected' : ''}${done ? ' task-row--done' : ''}`}
                  style={{ borderLeftColor: col }}
                  onClick={() => selectTask(t)}
                >
                  <div className="task-row__line1">
                    <span className="title">{t.title}</span>
                    {(hasTime || hasCount) && (
                      <span className="task-row__meta muted small">
                        {hasTime && fmtDuration(secs)}
                        {hasTime && hasCount && <span className="task-row__bullet"> • </span>}
                        {hasCount && `${k}/${t.sub_count}`}
                      </span>
                    )}
                  </div>
                  <div className="task-row__line2">
                    <span className="task-row__status" style={{ color: col }}>{t.status}</span>
                    {tags.length > 0 && (
                      <span className="task-row__tags">
                        {tags.slice(0, 3).map((tg) => (
                          <span key={tg.id} className="tag task-row__tag" style={{ background: tg.color || 'var(--panel2)' }}>{tg.text}</span>
                        ))}
                        {tags.length > 3 && <span className="muted small">+{tags.length - 3}</span>}
                      </span>
                    )}
                  </div>
                </li>
              )
            })}
          </ul>
        ) : (
          <p className="muted">{projects.length === 0 ? 'Создайте проект' : 'Выберите проект'}</p>
        )}
        <div className="resizer" onMouseDown={taskResize.onDown} />
      </div>

      {/* Колонка 2: подзадачи выбранной задачи */}
      <div className="panel col" style={{ width: subResize.width, position: 'relative', overflow: 'auto' }}>
        <div className="flex between" style={{ marginBottom: 8 }}>
          <strong className="col-head" style={{ margin: 0 }}>Подзадачи</strong>
          {selTaskId && <Button color="accent" icon="plus" label="Создать подзадачу" onClick={() => openAdd('sub')} />}
        </div>
        {!selTaskId && <p className="muted">Выберите задачу.</p>}
        {selTaskId && (
          <ul className="list">
            {subsOf(selTaskId).map((s) => {
              const col = statusColor(s.status, statuses)
              const done = isDone(s.status)
              let secs = s.total_seconds
              if (s.active_since) secs += Math.floor(Date.now() / 1000) - s.active_since
              const hasTime = secs > 0
              const cc = checkMap[String(s.id)]
              const hasCheck = !!cc && cc[1] > 0
              const ck = cc ? `${cc[0]}/${cc[1]}` : ''
              const showBullet = hasTime && hasCheck
              return (
                <li
                  key={s.id}
                  className={`row task-row${selSubId === s.id ? ' selected' : ''}${done ? ' task-row--done' : ''}`}
                  style={{ borderLeftColor: col }}
                  onClick={() => selectSub(s)}
                >
                  <div className="task-row__line1">
                    <span className="title">{s.title}</span>
                    <span className="task-row__meta muted small" style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                      {(hasTime || hasCheck) && (
                        <>
                          {hasTime && fmtDuration(secs)}
                          {showBullet && <span className="task-row__bullet"> • </span>}
                          {hasCheck && ck}
                        </>
                      )}
                      {runningId === s.id && <span className="running">●</span>}
                    </span>
                  </div>
                  <div className="task-row__line2">
                    <span className="task-row__status" style={{ color: col }}>{s.status}</span>
                  </div>
                </li>
              )
            })}
          </ul>
        )}
        <div className="resizer" onMouseDown={subResize.onDown} />
      </div>

      {/* Колонка 3: контент — задача сверху, разделитель, подзадача снизу */}
      <div className="panel col" style={{ flex: 1, overflow: 'auto' }}>
        {!selTaskId && !selSubId && <p className="muted">Выберите задачу или подзадачу.</p>}

        {selTaskId && (() => {
          const t = tasks.find((x) => x.id === selTaskId)!
          return (
            <TaskDetail
              task={t}
              detail={taskDetail}
              statuses={statuses}
              onError={onError}
              onStatus={changeStatus}
              onDesc={(v) => api.updateTaskDescription(selTaskId!, v).catch((e) => onError(String(e)))}
              onLinkAdd={async (n, u) => { await api.createTaskLink(selTaskId!, n, u); loadTaskDetail(selTaskId!) }}
              onLinkDel={async (id) => { await api.deleteTaskLink(id); loadTaskDetail(selTaskId!) }}
              onNewSub={async (title) => { await api.createSubtask(selTaskId!, title); reload() }}
              onDel={() => delTask(t)}
              onTagAdd={async (typeId, text, url) => { await api.createTag(selTaskId!, typeId, text, url); loadTaskDetail(selTaskId!) }}
              onTagDel={async (id) => { await api.deleteTag(id); loadTaskDetail(selTaskId!) }}
            />
          )
        })()}

        {selTaskId && selSubId && <hr className="divider" />}

        {selSubId && (() => {
          const s = subs.find((x) => x.id === selSubId)!
          return (
            <SubDetail
              sub={s}
              detail={subDetail}
              statuses={statuses}
              running={runningId === selSubId}
              onError={onError}
              onStatus={changeStatus}
              onToggleTimer={toggleTimer}
              onDesc={(v) => api.updateSubtaskDescription(selSubId!, v).catch((e) => onError(String(e)))}
              onLinkAdd={async (n, u) => { await api.createSubtaskLink(selSubId!, n, u); loadSubDetail(selSubId!) }}
              onLinkDel={async (id) => { await api.deleteSubtaskLink(id); loadSubDetail(selSubId!) }}
              onTimeEdit={async (id, s2, e2) => { await api.updateTimeEntry(id, s2, e2); loadSubDetail(selSubId!) }}
              onTimeDel={async (id) => { await api.deleteTimeEntry(id); loadSubDetail(selSubId!) }}
              onJournalAdd={async (text) => { await api.createJournal(selSubId!, text); loadSubDetail(selSubId!) }}
              onCheckToggle={async (id, st) => { await api.setChecklistStatus(id, st); loadSubDetail(selSubId!) }}
              onCheckAdd={async (text) => { await api.createChecklistItem(selSubId!, text); loadSubDetail(selSubId!) }}
              onCheckDel={async (id) => { await api.deleteChecklistItem(id); loadSubDetail(selSubId!) }}
              onDel={() => delSub(s)}
            />
          )
        })()}
      </div>


      {confirmNode}

      {addKind && (
        <Modal
          title={addKind === 'task' ? 'Новая задача' : 'Новая подзадача'}
          onClose={() => setAddKind(null)}
          footer={
            <>
              <Button variant="outline" onClick={() => setAddKind(null)}>Отмена</Button>
              <Button color="accent" icon="plus" disabled={!addTitle.trim()} onClick={submitAdd}>Создать</Button>
            </>
          }
        >
          <input
            autoFocus
            placeholder="Название"
            value={addTitle}
            style={{ width: '100%', minWidth: 360 }}
            onChange={(e) => setAddTitle(e.target.value)}
            onKeyDown={(e) => { if (e.key === 'Enter') submitAdd() }}
          />
        </Modal>
      )}
    </div>
  )
}

function statusColor(name: string, statuses: StatusDef[]): string {
  const s = statuses.find((x) => x.name === name)
  return s?.color || 'var(--grey)'
}

// --- Детальные панели ---

function TaskDetail(props: {
  task: Task
  detail: Detail
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
        <Button color="danger" variant="outline" icon="trash" label="Удалить задачу" onClick={props.onDel} />
      </div>
      <div>
        <div className="flex between"><strong>Описание</strong><Button color="accent" icon="save" onClick={() => props.onDesc(desc)}>Сохранить</Button></div>
        <textarea value={desc} onChange={(e) => setDesc(e.target.value)} />
      </div>
      <div className="flex" style={{ gap: 8 }}>
        <select value={st} onChange={(e) => setSt(e.target.value)}>
          <option value="">Сменить статус…</option>
          {props.statuses.map((s) => (<option key={s.id} value={s.name}>{s.name}</option>))}
        </select>
        {st && <input placeholder="Заметка" value={nt} onChange={(e) => setNt(e.target.value)} />}
        <Button color="accent" icon="check" disabled={!st} onClick={() => { props.onStatus(st, nt); setSt(''); setNt('') }}>Применить</Button>
      </div>
      <div>
        <div className="flex between"><strong>Подзадачи</strong></div>
        <div className="flex">
          <input placeholder="Название подзадачи" value={subTitle} onChange={(e) => setSubTitle(e.target.value)} />
          <Button color="accent" icon="plus" label="Добавить подзадачу" disabled={!subTitle.trim()} onClick={() => { props.onNewSub(subTitle.trim()); setSubTitle('') }} />
        </div>
      </div>
      <LinksBlock links={props.detail.links} onAdd={props.onLinkAdd} onDel={props.onLinkDel} />
      <TagsBlock tags={props.detail.tags} onAdd={props.onTagAdd} onDel={props.onTagDel} />
    </div>
  )
}

function SubDetail(props: {
  sub: Subtask
  detail: Detail
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
  onDel?: () => void
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
        <div className="flex" style={{ gap: 8 }}>
          {props.onDel && <Button color="danger" variant="outline" icon="trash" label="Удалить подзадачу" onClick={props.onDel} />}
          <Button color={props.running ? 'danger' : 'success'} icon={props.running ? 'pause' : 'play'} onClick={props.onToggleTimer}>
            {props.running ? 'Стоп' : 'Старт'}
          </Button>
        </div>
      </div>
      <div className="flex" style={{ gap: 8 }}>
        <select value={st} onChange={(e) => setSt(e.target.value)}>
          <option value="">Сменить статус…</option>
          {props.statuses.map((s) => (<option key={s.id} value={s.name}>{s.name}</option>))}
        </select>
        {st && <input placeholder="Заметка" value={nt} onChange={(e) => setNt(e.target.value)} />}
        <Button color="accent" icon="check" disabled={!st} onClick={() => { props.onStatus(st, nt); setSt(''); setNt('') }}>Применить</Button>
      </div>
      <div>
        <div className="flex between"><strong>Описание</strong><Button color="accent" icon="save" onClick={() => props.onDesc(desc)}>Сохранить</Button></div>
        <textarea value={desc} onChange={(e) => setDesc(e.target.value)} />
      </div>

      <div>
        <strong>Записи времени</strong>
        <ul className="list">
          {props.detail.time.map((t: TimeEntry) => (
            <li key={t.id} className="row">
              <span className="title">{fmtDateTime(t.started_at)} — {t.ended_at ? fmtDateTime(t.ended_at) : '…'}</span>
              <Button icon="edit" label="Редактировать" onClick={() => {
                const s = prompt('Начало (ISO)', t.started_at)
                if (!s) return
                const e = prompt('Конец (ISO, пусто = открыто)', t.ended_at ?? '')
                props.onTimeEdit(t.id, s, e || null)
              }} />
              <Button color="danger" variant="outline" icon="trash" label="Удалить" onClick={() => props.onTimeDel(t.id)} />
            </li>
          ))}
        </ul>
      </div>

      <div>
        <strong>Журнал</strong>
        <div className="flex">
          <input placeholder="Новая запись" value={jt} onChange={(e) => setJt(e.target.value)} />
          <Button color="accent" icon="plus" label="Добавить запись" disabled={!jt.trim()} onClick={() => { props.onJournalAdd(jt.trim()); setJt('') }} />
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
          <Button color="accent" icon="plus" label="Добавить пункт" disabled={!ct.trim()} onClick={() => { props.onCheckAdd(ct.trim()); setCt('') }} />
        </div>
        <ul className="list">
          {props.detail.checklist.map((c: ChecklistItem) => (
            <li key={c.id} className="row">
              <select value={c.status} onChange={(e) => props.onCheckToggle(c.id, e.target.value)}>
                {checkStates.map((s) => (<option key={s} value={s}>{s}</option>))}
              </select>
              <span className="title">{c.text}</span>
              <Button color="danger" variant="outline" icon="trash" label="Удалить" onClick={() => props.onCheckDel(c.id)} />
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
        <Button color="accent" icon="plus" label="Добавить ссылку" disabled={!ln.url.trim()} onClick={() => { onAdd(ln.name, ln.url); setLn({ name: '', url: '' }) }} />
      </div>
      <ul className="list">
        {links.map((l) => (
          <li key={l.id} className="row">
            <a href={l.url} target="_blank" rel="noreferrer">{l.name || l.url}</a>
            <Button color="danger" variant="outline" icon="trash" label="Удалить ссылку" onClick={() => onDel(l.id)} />
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
        <Button color="accent" icon="plus" label="Добавить тег" disabled={!t.typeId || !t.text.trim()} onClick={() => { onAdd(t.typeId, t.text, t.url); setT({ typeId: 0, text: '', url: '' }) }} />
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
