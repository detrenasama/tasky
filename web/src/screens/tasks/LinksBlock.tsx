import { useState } from 'react'
import type { Link } from '../../types'
import { Button } from '../../ui'

export function LinksBlock({ links, onAdd, onDel }: { links: Link[]; onAdd: (n: string, u: string) => void; onDel: (id: number) => void }) {
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
