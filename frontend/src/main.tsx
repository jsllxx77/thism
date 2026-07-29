import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import './index.css'
import App from './App.tsx'
import { LanguageProvider } from './i18n/language'
import { AppThemeProvider } from './theme/theme'
import { ThemeModeProvider } from './theme/mode'
import { DEFAULT_SHADCN_PLUGIN } from './theme-plugin/default-shadcn'
import { ThemePluginRuntimeProvider } from './theme-plugin/runtime'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <LanguageProvider>
      <ThemeModeProvider>
        <AppThemeProvider>
          <ThemePluginRuntimeProvider plugin={DEFAULT_SHADCN_PLUGIN}>
            <BrowserRouter>
              <App />
            </BrowserRouter>
          </ThemePluginRuntimeProvider>
        </AppThemeProvider>
      </ThemeModeProvider>
    </LanguageProvider>
  </StrictMode>,
)
