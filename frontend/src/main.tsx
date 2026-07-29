import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import './index.css'
import App from './App.tsx'
import { LanguageProvider } from './i18n/language'
import { AppThemeProvider } from './theme/theme'
import { ThemeModeProvider } from './theme/mode'
import { ConfiguredThemePluginRuntime } from './theme-plugin/configured-runtime'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <LanguageProvider>
      <ThemeModeProvider>
        <AppThemeProvider>
          <ConfiguredThemePluginRuntime>
            <BrowserRouter>
              <App />
            </BrowserRouter>
          </ConfiguredThemePluginRuntime>
        </AppThemeProvider>
      </ThemeModeProvider>
    </LanguageProvider>
  </StrictMode>,
)
