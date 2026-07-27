import Dexie, { type Table } from 'dexie'

import type { ImportItemRow, ImportSessionRow } from './state'

export class ImporterDatabase extends Dexie {
  sessions!: Table<ImportSessionRow, string>
  items!: Table<ImportItemRow, [string, string]>

  constructor(name = 'craftsky-instagram-importer') {
    super(name)
    this.version(1).stores({
      sessions:
        'id, did, manifestFingerprint, status, createdAt, updatedAt',
      items: '[sessionId+itemKey], sessionId, status, rkey',
    })
  }
}

export const importerDatabase = new ImporterDatabase()
