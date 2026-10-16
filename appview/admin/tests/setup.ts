import { afterEach, vi } from 'vitest'
import { cleanup } from '@testing-library/react'
afterEach(() => { cleanup(); vi.unstubAllGlobals(); window.history.replaceState(null, '', '/admin/') })
