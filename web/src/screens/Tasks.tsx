import { useEffect, useState } from 'react'
import { api } from '../api'
import type {
  Project,
  Task,
  Subtask,
  Tag,
  TagType,
  StatusDef,
} from '../types'
import { useConfirm, useColumnWidth } from '../ui'
import '../ui/statusButton.css'
import '../ui/tagBar.css'
import { TaskDetail } from './tasks/TaskDetail'
import { SubDetail } from './tasks/SubDetail'
import { TaskListColumn } from './tasks/TaskListColumn'
import { SubListColumn } from './tasks/SubListColumn'
import { TaskGhost } from './tasks/TaskGhost'
import { StatusNoteModal, AddModal } from './tasks/TaskModals'
import { useTaskDrag } from './tasks/useTaskDrag'
import { Detail, EMPTY } from './tasks/types'
import './tasks/taskRow.css'
import './tasks/drag.css'

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
  const [pendingStatus, setPendingStatus] = useState<{ kind: 'task' | 'subtask'; id: number; to: string; prompt: string } | null>(null)
  const [pendingNote, setPendingNote] = useState('')
  const [tagTypes, setTagTypes] = useState<TagType[]>([])

  const taskResize = useColumnWidth(320, 'tasky.col.tasks')
  const subResize = useColumnWidth(300, 'tasky.col.subs')

  useEffect(() => {
    api.statuses().then(setStatuses).catch((e) => onError(String(e)))
    api.tagTypes().then(setTagTypes).catch((e) => onError(String(e)))
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

  const execStatus = async (kind: 'task' | 'subtask', id: number, to: string, note: string) => {
    try {
      await api.setStatus(kind, id, to, note)
      reload()
      if (kind === 'subtask') loadSubDetail(id)
      else loadTaskDetail(id)
    } catch (e) {
      onError(String(e))
    }
  }

  const requestStatus = (kind: 'task' | 'subtask', id: number, to: string) => {
    const def = statuses.find((s) => s.name === to)
    if (def?.note_prompt) {
      setPendingNote('')
      setPendingStatus({ kind, id, to, prompt: def.note_prompt })
    } else {
      execStatus(kind, id, to, '')
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

  const {
    taskListRef,
    subListRef,
    dragTask,
    dragSub,
    ghost,
    suppressClick,
    displayTasks,
    displaySubsFor,
    onTaskPointerDown,
    onSubPointerDown,
  } = useTaskDrag({ tasks, subs, selTaskId, setTasks, setSubs, setSelTaskId, onError, reload })

  return (
    <div className="flex" style={{ alignItems: 'stretch', height: '100%' }}>
      <TaskListColumn
        projects={projects}
        proj={proj}
        tasks={displayTasks as Task[]}
        statuses={statuses}
        tagsMap={tagsMap}
        subs={subs}
        selTaskId={selTaskId}
        dragTask={dragTask}
        dragSub={dragSub}
        ghost={ghost}
        taskListRef={taskListRef}
        suppressClick={suppressClick}
        onProjectChange={openProject}
        onAddTask={() => openAdd('task')}
        onSelectTask={selectTask}
        onTaskPointerDown={onTaskPointerDown}
        taskResize={taskResize}
      />
      <SubListColumn
        selTaskId={selTaskId}
        subs={subs}
        displaySubsFor={displaySubsFor}
        statuses={statuses}
        checkMap={checkMap}
        runningId={runningId}
        selSubId={selSubId}
        dragSub={dragSub}
        ghost={ghost}
        subListRef={subListRef}
        suppressClick={suppressClick}
        onAddSub={() => openAdd('sub')}
        onSelectSub={selectSub}
        onSubPointerDown={onSubPointerDown}
        subResize={subResize}
      />
      <div className="panel col" style={{ flex: 1, overflow: 'auto' }}>
        {!selTaskId && !selSubId && <p className="muted">Выберите задачу или подзадачу.</p>}
        {selSubId
          ? (() => {
              const s = subs.find((x) => x.id === selSubId)!
              return (
                <SubDetail
                  sub={s}
                  detail={subDetail}
                  statuses={statuses}
                  running={runningId === selSubId}
                  onStatusPick={(to) => requestStatus('subtask', s.id, to)}
                  onToggleTimer={toggleTimer}
                  onDesc={async (v) => { await api.updateSubtaskDescription(selSubId!, v); loadSubDetail(selSubId!) }}
                  onLinkAdd={async (n, u) => { await api.createSubtaskLink(selSubId!, n, u); loadSubDetail(selSubId!) }}
                  onLinkEdit={async (id, n, u) => { await api.updateSubtaskLink(id, n, u); loadSubDetail(selSubId!) }}
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
            })()
          : selTaskId
            ? (() => {
                const t = tasks.find((x) => x.id === selTaskId)!
                return (
                  <TaskDetail
                    task={t}
                    detail={taskDetail}
                    statuses={statuses}
                    tagTypes={tagTypes}
                    onStatusPick={(to) => requestStatus('task', t.id, to)}
                    onDesc={async (v) => { await api.updateTaskDescription(selTaskId!, v); loadTaskDetail(selTaskId!) }}
                    onLinkAdd={async (n, u) => { await api.createTaskLink(selTaskId!, n, u); loadTaskDetail(selTaskId!) }}
                    onLinkEdit={async (id, n, u) => { await api.updateTaskLink(id, n, u); loadTaskDetail(selTaskId!) }}
                    onLinkDel={async (id) => { await api.deleteTaskLink(id); loadTaskDetail(selTaskId!) }}
                    onDel={() => delTask(t)}
                    onTagAdd={async (typeId, text, url) => { await api.createTag(selTaskId!, typeId, text, url); loadTaskDetail(selTaskId!); api.tagsByProject(t.project_id).then(setTagsMap).catch(()=>{}) }}
                    onTagEdit={async (id, typeId, text, url) => { await api.updateTag(id, typeId, text, url); loadTaskDetail(selTaskId!); api.tagsByProject(t.project_id).then(setTagsMap).catch(()=>{}) }}
                    onTagDel={async (id) => { await api.deleteTag(id); loadTaskDetail(selTaskId!); api.tagsByProject(t.project_id).then(setTagsMap).catch(()=>{}) }}
                  />
                )
              })()
            : null}
      </div>

      <StatusNoteModal pendingStatus={pendingStatus} pendingNote={pendingNote} setPendingNote={setPendingNote} setPendingStatus={setPendingStatus} execStatus={execStatus} />
      {confirmNode}
      <TaskGhost ghost={ghost} tasks={tasks} subs={subs} tagsMap={tagsMap} checkMap={checkMap} statuses={statuses} />
      <AddModal addKind={addKind} addTitle={addTitle} setAddKind={setAddKind} setAddTitle={setAddTitle} submitAdd={submitAdd} />
    </div>
  )
}
