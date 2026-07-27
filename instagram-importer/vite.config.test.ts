import { describe, expect, it } from 'vitest'

import config from './vite.config'

describe('Vite development server', () => {
  it('listens on the IPv4 loopback address used by local OAuth', () => {
    expect(config).toMatchObject({
      server: {
        host: '127.0.0.1',
      },
    })
  })
})
