import { useState } from 'react'
import type { Subtask, Link, TimeEntry, JournalEntry, ChecklistItem, StatusDef } from '../../types'
import { fmtDateTime } from '../../fmt'
import { Button } from '../../ui'
import { StatusButton } from '../../ui/statusButton'
import { DescriptionBlock } from '../../ui/descriptionBlock'
import { LinksBlock } from './LinksBlock'

type Detail = {
  description: string
  links: Link[]
  time: TimeEntry[]
  journal: JournalEntry[]
  checklist: ChecklistItem[]
}

export function SubDetail(props: {
  sub: Subtask
  detail: Detail
  statuses: StatusDef[]
  running: boolean
  onStatusPick: (to: string) => void
  onToggleTimer: () => void
  onDesc: (v: string) => void
  onLinkAdd: (n: string, u: string) => void
  onLinkEdit: (id: number, n: string, u: string) => void
  onLinkDel: (id: number) => void
  onTimeEdit: (id: number, started: string, ended: string | null) => void
  onTimeDel: (id: number) => void
  onJournalAdd: (text: string) => void
  onCheckToggle: (id: number, status: string) => void
  onCheckAdd: (text: string) => void
  onCheckDel: (id: number) => void
  onDel?: () => void
}) {
  const [jt, setJt] = useState('')
  const [ct, setCt] = useState('')

  const checkStates = ['new', 'in_progress', 'done', 'cancelled']

  return (
    <div className="col">
      <div className="flex between">
        <div className="flex" style={{ gap: 8, alignItems: 'center' }}>
          <StatusButton value={props.sub.status} statuses={props.statuses} onSelect={props.onStatusPick} />
          <h2 style={{ margin: 0 }}>{props.sub.title}</h2>
        </div>
        <div className="flex" style={{ gap: 8 }}>
          {props.onDel && <Button color="danger" variant="outline" icon="trash" label="Удалить подзадачу" onClick={props.onDel} />}
          <Button color={props.running ? 'danger' : 'success'} icon={props.running ? 'pause' : 'play'} onClick={props.onToggleTimer}>
            {props.running ? 'Стоп' : 'Старт'}
          </Button>
        </div>
      </div>
      <DescriptionBlock value={props.detail.description} onSave={props.onDesc} />

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

      <LinksBlock links={props.detail.links} onAdd={props.onLinkAdd} onEdit={props.onLinkEdit} onDel={props.onLinkDel} />
    </div>
  )
}
