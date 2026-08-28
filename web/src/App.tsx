import { useEffect, useState } from 'react'
import { api } from './api'
import type { StatusPoll } from './types'
import { fmtDuration } from './fmt'
import { useToast } from './ui'
import Projects from './screens/Projects'
import Tasks from './screens/Tasks'
import Reports from './screens/Reports'
import Settings from './screens/Settings'

type Screen = 'projects' | 'tasks' | 'reports' | 'settings'

export default function App() {
  const [screen, setScreen] = useState<Screen>('tasks')
  const [poll, setPoll] = useState<StatusPoll | null>(null)
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
    return () => {
      alive = false
      clearInterval(h)
    }
  }, [])

  const tabs: { id: Screen; label: string }[] = [
    { id: 'projects', label: 'Проекты' },
    { id: 'tasks', label: 'Задачи' },
    { id: 'reports', label: 'Отчёты' },
    { id: 'settings', label: 'Настройки' },
  ]

  return (
    <div className="app">
      <div className="sidebar">
        {tabs.map((t) => (
          <div
            key={t.id}
            className={`tab ${screen === t.id ? 'active' : ''}`}
            onClick={() => setScreen(t.id)}
          >
            {t.label}
          </div>
        ))}
      </div>
      <div className="content">
        <div className="topbar">
          <strong>Tasky</strong>
          <span className="muted">Сегодня: {fmtDuration(poll?.today_seconds ?? 0)}</span>
          {poll?.running && <span className="running">● {poll.running.title}</span>}
        </div>
        <div className="main" style={{ display: 'block' }}>
          {screen === 'projects' && <Projects onError={setToast} />}
          {screen === 'tasks' && <Tasks onError={setToast} />}
          {screen === 'reports' && <Reports onError={setToast} />}
          {screen === 'settings' && <Settings onError={setToast} />}
        </div>
      </div>
      {toast && <div className="toast">{toast}</div>}
    </div>
  )
}
