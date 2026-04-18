/// <reference types="vitest" />
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { VitePWA } from 'vite-plugin-pwa'
import path from 'path'

export default defineConfig({
  define: {
    __BUILD_NUMBER__: JSON.stringify(process.env.BUILD_NUMBER || 'dev'),
    __BUILD_TIME__: JSON.stringify(new Date().toISOString()),
    __COMMIT_SHA__: JSON.stringify(process.env.COMMIT_SHA || 'local'),
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    css: false,
  },
  plugins: [
    react(),
    tailwindcss(),
    VitePWA({
      registerType: 'autoUpdate',
      workbox: {
        // 새 SW 가 install 되면 waiting 단계 없이 즉시 activate.
        // 새 코드가 더 빨리 사용자에게 도달하지만, 활성 탭이 다른 SW 를 쓰는 동안
        // 일시적인 불일치가 생길 수 있어서 VersionWatcher 가 토스트로 reload 안내.
        skipWaiting: true,
        clientsClaim: true,
        cleanupOutdatedCaches: true,
        globPatterns: ['**/*.{js,css,html,ico,png,svg,woff2}'],
        importScripts: ['/sw-push.js'],
        navigateFallbackDenylist: [/^\/docs/, /^\/api\//, /^\/uploads\//],
        runtimeCaching: [
          {
            urlPattern: /^\/api\//,
            handler: 'NetworkFirst',
            options: { cacheName: 'api-cache', expiration: { maxEntries: 50 } }
          }
        ]
      },
      manifest: {
        name: 'EarnLearning',
        short_name: 'EarnLearning',
        description: '스타트업을 위한 코딩입문 LMS',
        theme_color: '#00643E',
        background_color: '#F8F7F4',
        display: 'standalone',
        icons: [
          { src: '/pwa-192x192.png', sizes: '192x192', type: 'image/png' },
          { src: '/pwa-512x512.png', sizes: '512x512', type: 'image/png', purpose: 'any' },
          { src: '/apple-touch-icon.png', sizes: '180x180', type: 'image/png' },
        ],
      }
    })
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    proxy: {
      '/api/ws': { target: 'http://localhost:8080', ws: true },
      '/api': 'http://localhost:8080',
      '/ws': { target: 'ws://localhost:8080', ws: true },
      '/uploads': 'http://localhost:8080',
      '/docs': 'http://localhost:8080',
    }
  }
})
