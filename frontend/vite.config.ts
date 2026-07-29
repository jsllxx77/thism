import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import path from 'path'
import { getManualChunkName } from './build/chunks'
import { filterModulePreloadDependencies } from './build/modulePreload'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@thism/theme-sdk/testing': path.resolve(__dirname, './src/theme-sdk/testing.ts'),
      '@thism/theme-sdk': path.resolve(__dirname, './src/theme-sdk/index.ts'),
      '@': path.resolve(__dirname, './src'),
    },
  },
  build: {
    modulePreload: {
      resolveDependencies: (url, deps, context) => filterModulePreloadDependencies(url, deps, context),
    },
    rollupOptions: {
      output: {
        manualChunks(id) {
          return getManualChunkName(id)
        },
      },
    },
  },
  server: {
    proxy: {
      '/api': 'http://localhost:12026',
      '/ws': { target: 'ws://localhost:12026', ws: true },
      '/install.sh': 'http://localhost:12026',
      '/dl': 'http://localhost:12026',
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    passWithNoTests: true,
  },
})
