import React from 'react'
import { createRoot } from 'react-dom/client'
import App from './App'
import './styles/tokens.css'
import './styles/reset.css'
import './styles/layout.css'
import './styles/topbar.css'
import './styles/list.css'
import './styles/forms.css'
import './styles/utilities.css'

createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
