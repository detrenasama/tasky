import type { Subtask, Link, TimeEntry, JournalEntry, ChecklistItem, StatusDef } from '../../types'
import { Button } from '../../ui'
import { StatusButton } from '../../ui/statusButton'
import { DescriptionBlock } from '../../ui/descriptionBlock'
import { LinksBlock } from './LinksBlock'
import { JournalBlock } from './JournalBlock'
import { ChecklistBlock } from './ChecklistBlock'
import { TimeEntriesBlock } from './TimeEntriesBlock'
import './subDetail.css'

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
  onCheckEdit: (id: number, text: string) => void
  onDel?: () => void
}) {
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

      <div className="subdetail-split">
        <div className="subdetail-col">
          <LinksBlock links={props.detail.links} onAdd={props.onLinkAdd} onEdit={props.onLinkEdit} onDel={props.onLinkDel} />
        </div>
        <div className="subdetail-col">
          <JournalBlock entries={props.detail.journal} onAdd={props.onJournalAdd} />
        </div>
      </div>

      <div className="subdetail-split">
        <div className="subdetail-col">
          <ChecklistBlock
            items={props.detail.checklist}
            onToggle={props.onCheckToggle}
            onAdd={props.onCheckAdd}
            onDel={props.onCheckDel}
            onEdit={props.onCheckEdit}
          />
        </div>
        <div className="subdetail-col">
          <TimeEntriesBlock
            entries={props.detail.time}
            subtaskTitle={props.sub.title}
            onEdit={props.onTimeEdit}
            onDel={props.onTimeDel}
          />
        </div>
      </div>
    </div>
  )
}
