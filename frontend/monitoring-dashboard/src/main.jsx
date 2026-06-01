import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App.jsx'
import faviconUrl from '../assets/favicon.ico'

const faviconLink =
  document.querySelector("link[rel*='icon']") ??
  document.head.appendChild(document.createElement('link'))

faviconLink.rel = 'icon'
faviconLink.type = 'image/x-icon'
faviconLink.href = faviconUrl

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <App />
  </StrictMode>
)
