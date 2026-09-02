import type { Task, TagType, Link, Tag, StatusDef } from '../../types'
import { Button } from '../../ui'
import { StatusButton } from '../../ui/statusButton'
import { TagBar } from '../../ui/tagBar'
import { DescriptionBlock } from '../../ui/descriptionBlock'
import { LinksBlock } from './LinksBlock'

type Detail = {
  description: string
  links: Link[]
  tags: Tag[]
}

export function TaskDetail(props: {
  task: Task
  detail: Detail
  statuses: StatusDef[]
  tagTypes: TagType[]
  onStatusPick: (to: string) => void
  onDesc: (v: string) => void
  onLinkAdd: (n: string, u: string) => void
  onLinkEdit: (id: number, n: string, u: string) => void
  onLinkDel: (id: number) => void
  onDel: () => void
  onTagAdd: (typeId: number, text: string, url: string) => void
  onTagEdit: (id: number, typeId: number, text: string, url: string) => void
  onTagDel: (id: number) => void
}) {

  return (
    <div className="col">
      <div className="flex between">
        <div className="flex" style={{ gap: 8, alignItems: 'center' }}>
          <StatusButton value={props.task.status} statuses={props.statuses} onSelect={props.onStatusPick} />
          <h2 style={{ margin: 0 }}>{props.task.title}</h2>
        </div>
        <Button color="danger" variant="outline" icon="trash" label="Удалить задачу" onClick={props.onDel} />
      </div>
      <TagBar tags={props.detail.tags} tagTypes={props.tagTypes} onAdd={props.onTagAdd} onEdit={props.onTagEdit} onDelete={props.onTagDel} />
      <DescriptionBlock value={props.detail.description} onSave={props.onDesc} />
      <LinksBlock links={props.detail.links} onAdd={props.onLinkAdd} onEdit={props.onLinkEdit} onDel={props.onLinkDel} />
    </div>
  )
}
