import { useEffect, useState } from 'react'
import { api } from './api'
import type { StatusPoll } from './types'
import { fmtDuration } from './fmt'
import { useToast, Modal, Button, ErrorBoundary } from './ui'
import Projects from './screens/Projects'
import Tasks from './screens/Tasks'
import Reports from './screens/Reports'
import Settings from './screens/Settings'

type Screen = 'tasks' | 'projects' | 'reports' | 'settings'

export default function App() {
  const [screen, setScreen] = useState<Screen>('tasks')
  const [poll, setPoll] = useState<StatusPoll | null>(null)
  const [version, setVersion] = useState('')
  const [toast, setToast] = useToast()

  useEffect(() => {
    let alive = true
    const tick = async () => {
      try {
        const p = await api.statusPoll()
        if (alive) setPoll(p)
      } catch (e) {
        if (alive) setToast(String(e))
      }
    }
    tick()
    const h = setInterval(tick, 2000)
    api.version().then((v) => setVersion(v.version)).catch(() => {})
    return () => {
      alive = false
      clearInterval(h)
    }
  }, [])

  const tabs: { id: Screen; label: string }[] = [
    { id: 'tasks', label: 'Задачи' },
    { id: 'projects', label: 'Проекты' },
    { id: 'reports', label: 'Отчёты' },
    { id: 'settings', label: 'Настройки' },
  ]

  return (
    <div className="app">
      <header className="topbar">
        <div className="brand">
          Tasky <span className="muted">v{version}</span>
        </div>
        <nav className="nav">
          {tabs.map((t) => (
            <span
              key={t.id}
              className={`nav-tab ${screen === t.id ? 'active' : ''}`}
              onClick={() => setScreen(t.id)}
            >
              {t.label}
            </span>
          ))}
        </nav>
        <div className="topright">
          <span className="muted">Сегодня: {fmtDuration(poll?.today_seconds ?? 0)}</span>
          {poll?.running && <span className="running">● {poll.running.title}</span>}
        </div>
      </header>
      <div className="main" style={{ display: 'block' }}>
        <ErrorBoundary>
          {screen === 'projects' && <Projects onError={setToast} />}
          {screen === 'tasks' && <Tasks onError={setToast} />}
          {screen === 'reports' && <Reports onError={setToast} />}
          {screen === 'settings' && <Settings onError={setToast} />}
        </ErrorBoundary>
      </div>
      {toast && (
        <Modal
          title="Ошибка"
          onClose={() => setToast(null)}
          footer={<Button className="primary" onClick={() => setToast(null)}>OK</Button>}
        >
          <p className="muted">{toast}</p>
        </Modal>
      )}
    </div>
  )
}
