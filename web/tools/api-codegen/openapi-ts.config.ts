import { defineConfig } from '@hey-api/openapi-ts'

export default defineConfig({
  input: '../../shared/api.openapi.json',
  output: {
    path: '../../shared/api',
    // Generation must work before Nuxt dependencies are installed, so do not
    // discover web/tsconfig.json (which intentionally extends .nuxt).
    tsConfigPath: null,
    header: [
      '// Generated from shared/api.openapi.json by @hey-api/openapi-ts.',
      '// Do not edit by hand; run `make gen-api-client`.',
    ],
  },
  plugins: ['@hey-api/typescript'],
})
