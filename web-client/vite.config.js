import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// Em desenvolvimento, /api e /swagger são encaminhados ao API Gateway local.
// Em produção (Docker), o Nginx faz o mesmo papel (ver nginx.conf).
const gateway = process.env.GATEWAY_URL ?? 'http://localhost:8080'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: 5173,
    proxy: {
      '/api': { target: gateway, changeOrigin: true },
      '/swagger': { target: gateway, changeOrigin: true },
    },
  },
  test: {
    environment: 'node',
  },
})
