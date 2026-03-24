import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { defineConfig } from 'vitepress'

const docsDir = dirname(fileURLToPath(import.meta.url))
const phenodocsRoot = resolve(docsDir, '../../../phenodocs')
const phenodocsTheme = resolve(phenodocsRoot, '.vitepress/theme/index.ts')

export default defineConfig({
  title: 'Bifrost Extensions',
  description: 'Clean extension layer for the Bifrost LLM gateway',
  lastUpdated: true,
  ignoreDeadLinks: true,

  vite: {
    resolve: {
      alias: {
        '@phenodocs-theme': phenodocsTheme,
      },
    },
    server: {
      fs: {
        allow: [phenodocsRoot],
      },
    },
  },
})
