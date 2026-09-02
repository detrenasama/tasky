import { useEffect, useState } from 'react'
import { api } from '../api'
import type { Project, Link } from '../types'
import { Button, Modal, useConfirm, MenuState, ContextMenu } from '../ui'
import { DescriptionBlock } from '../ui/descriptionBlock'

export default function Projects({ onError }: { onError: (m: string) => void }) {
  const [projects, setProjects] = useState<Project[]>([])
  const [sel, setSel] = useState<Project | null>(null)
  const [desc, setDesc] = useState('')
  const [links, setLinks] = useState<Link[]>([])
  const [menu, setMenu] = useState<MenuState | null>(null)
  const [creating, setCreating] = useState(false)
  const [newName, setNewName] = useState('')
  const [editingLink, setEditingLink] = useState<Link | null>(null)
  const [linkForm, setLinkForm] = useState({ name: '', url: '' })
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

  const saveLink = async () => {
    if (!editingLink) return
    try {
      await api.updateProjectLink(editingLink.id, linkForm.name, linkForm.url)
      setEditingLink(null)
      if (sel) select(sel)
    } catch (e) {
      onError(String(e))
    }
  }

  const delLink = async (l: Link) => {
    if (!(await confirm('Удалить ссылку?'))) return
    try {
      await api.deleteProjectLink(l.id)
      if (sel) select(sel)
    } catch (e) {
      onError(String(e))
    }
  }

  const openNewLink = () => {
    setLinkForm({ name: '', url: '' })
    setEditingLink({ id: 0, owner_id: sel!.id, name: '', url: '', created_at: '' })
  }

  const createLink = async () => {
    if (!sel) return
    try {
      await api.createProjectLink(sel.id, linkForm.name, linkForm.url)
      setEditingLink(null)
      select(sel)
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
            <div>
              <div className="flex between">
                <strong>Ссылки</strong>
                <Button color="accent" icon="plus" label="Добавить ссылку" onClick={openNewLink} />
              </div>
              <ul className="list">
                {links.map((l) => (
                  <li
                    key={l.id}
                    className="row"
                    onContextMenu={(e) => {
                      e.preventDefault()
                      setMenu({
                        x: e.clientX,
                        y: e.clientY,
                        items: [
                          {
                            label: 'Изменить',
                            onClick: () => {
                              setLinkForm({ name: l.name, url: l.url })
                              setEditingLink(l)
                            },
                          },
                          { label: 'Удалить', danger: true, onClick: () => delLink(l) },
                        ],
                      })
                    }}
                  >
                    <a href={l.url} target="_blank" rel="noreferrer" onClick={(e) => e.stopPropagation()}>
                      {l.name || l.url}
                    </a>
                  </li>
                ))}
              </ul>
            </div>
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

      {editingLink && (
        <Modal
          title={editingLink.id ? 'Изменить ссылку' : 'Новая ссылка'}
          onClose={() => setEditingLink(null)}
          footer={
            <>
              <Button variant="outline" onClick={() => setEditingLink(null)}>Отмена</Button>
              <Button color="accent" icon="save" onClick={editingLink.id ? saveLink : createLink}>
                Сохранить
              </Button>
            </>
          }
        >
          <div className="col">
            <input
              autoFocus
              placeholder="Название (необязательно)"
              value={linkForm.name}
              onChange={(e) => setLinkForm({ ...linkForm, name: e.target.value })}
            />
            <input
              placeholder="URL"
              value={linkForm.url}
              onChange={(e) => setLinkForm({ ...linkForm, url: e.target.value })}
            />
          </div>
        </Modal>
      )}

      <ContextMenu state={menu} onClose={() => setMenu(null)} />
    </div>
  )
}
