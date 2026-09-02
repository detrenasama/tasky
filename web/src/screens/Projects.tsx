import { useEffect, useState } from 'react'
import { api } from '../api'
import type { Project, Link } from '../types'
import { Button, Modal, useConfirm } from '../ui'
import { DescriptionBlock } from '../ui/descriptionBlock'
import { LinksBlock } from '../ui/linksBlock'

export default function Projects({ onError }: { onError: (m: string) => void }) {
  const [projects, setProjects] = useState<Project[]>([])
  const [sel, setSel] = useState<Project | null>(null)
  const [desc, setDesc] = useState('')
  const [links, setLinks] = useState<Link[]>([])
  const [creating, setCreating] = useState(false)
  const [newName, setNewName] = useState('')
  const confirm = useConfirm()[0]

  const load = () => {
    api
      .projects()
      .then(setProjects)
      .catch((e) => onError(String(e)))
  }
  useEffect(load, [])

  const select = (p: Project) => {
    setSel(p)
    api.projectDescription(p.id).then((d) => setDesc(d.description)).catch((e) => onError(String(e)))
    api.projectLinks(p.id).then(setLinks).catch((e) => onError(String(e)))
  }

  const create = async () => {
    if (!newName.trim()) return
    try {
      const p = await api.createProject(newName.trim())
      setNewName('')
      setCreating(false)
      load()
      setSel(p)
    } catch (e) {
      onError(String(e))
    }
  }

  const del = async (p: Project) => {
    if (!(await confirm(`Удалить проект «${p.name}»?`))) return
    try {
      await api.deleteProject(p.id)
      setSel(null)
      load()
    } catch (e) {
      onError(String(e))
    }
  }

  const saveDesc = async (v: string) => {
    if (!sel) return
    try {
      await api.updateProjectDescription(sel.id, v)
      setDesc(v)
    } catch (e) {
      onError(String(e))
    }
  }



  return (
    <div className="flex" style={{ alignItems: 'stretch', height: '100%' }}>
      <div className="panel col" style={{ width: 280, overflow: 'auto' }}>
        <div className="toolbar">
          <Button color="accent" icon="plus" label="Создать проект" onClick={() => setCreating(true)} />
        </div>
        <ul className="list">
          {projects.map((p) => (
            <li
              key={p.id}
              className={`row ${sel?.id === p.id ? 'selected' : ''}`}
              onClick={() => select(p)}
            >
              <span className="title">{p.name}</span>
            </li>
          ))}
        </ul>
      </div>

      <div className="panel col" style={{ flex: 1, overflow: 'auto' }}>
        {!sel && <p className="muted">Выберите проект слева.</p>}
        {sel && (
          <>
            <div className="flex between">
              <h2 style={{ margin: 0 }}>{sel.name}</h2>
              <Button color="danger" variant="outline" icon="trash" label="Удалить проект" onClick={() => del(sel)} />
            </div>
            <DescriptionBlock value={desc} onSave={saveDesc} />
            <LinksBlock
              links={links}
              onAdd={async (n, u) => { if (!sel) return; await api.createProjectLink(sel.id, n, u); select(sel) }}
              onEdit={async (id, n, u) => { await api.updateProjectLink(id, n, u); if (sel) select(sel) }}
              onDel={async (id) => { await api.deleteProjectLink(id); if (sel) select(sel) }}
            />
          </>
        )}
      </div>

      {creating && (
        <Modal
          title="Новый проект"
          onClose={() => setCreating(false)}
          footer={
            <>
              <Button variant="outline" onClick={() => setCreating(false)}>Отмена</Button>
              <Button color="accent" icon="plus" onClick={create}>
                Создать
              </Button>
            </>
          }
        >
          <input
            autoFocus
            placeholder="Название"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
          />
        </Modal>
      )}


    </div>
  )
}
