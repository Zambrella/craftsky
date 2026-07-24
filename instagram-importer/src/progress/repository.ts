import type { ImporterDatabase } from './database'
import type { ImportItemRow, ImportSessionRow } from './state'
import { isSafeErrorCode } from '../privacy/safeDiagnostics'

export interface StoredImport {
  readonly session: ImportSessionRow
  readonly items: readonly ImportItemRow[]
}

export class ProgressRepository {
  constructor(private readonly database: ImporterDatabase) {}

  async createSession(session: ImportSessionRow): Promise<void> {
    await this.database.sessions.add(session)
  }

  async createImport(
    session: ImportSessionRow,
    items: readonly ImportItemRow[],
  ): Promise<void> {
    await this.database.transaction(
      'rw',
      this.database.sessions,
      this.database.items,
      async () => {
        await this.database.sessions.add(session)
        await this.database.items.bulkAdd([...items])
      },
    )
  }

  async putSession(session: ImportSessionRow): Promise<void> {
    await this.database.sessions.put(session)
  }

  async putItem(item: ImportItemRow): Promise<void> {
    await this.database.items.put(item)
  }

  async putItems(items: readonly ImportItemRow[]): Promise<void> {
    await this.database.items.bulkPut([...items])
  }

  async load(sessionId: string): Promise<StoredImport | null> {
    const session = await this.database.sessions.get(sessionId)
    if (!session) return null
    const rawItems = await this.database.items
      .where('sessionId')
      .equals(sessionId)
      .toArray()
    const items = rawItems.map((item) => ({
      ...item,
      safeCode:
        item.safeCode === undefined || isSafeErrorCode(item.safeCode)
          ? item.safeCode
          : undefined,
    }))
    return { session, items }
  }

  async findResumable(
    did: string,
    manifestFingerprint: string,
  ): Promise<StoredImport | null> {
    const sessions = await this.database.sessions
      .where('did')
      .equals(did)
      .filter(
        (session) =>
          session.manifestFingerprint === manifestFingerprint,
      )
      .reverse()
      .sortBy('updatedAt')
    const session = sessions.at(0)
    return session ? this.load(session.id) : null
  }

  async clearSession(sessionId: string): Promise<void> {
    await this.database.transaction(
      'rw',
      this.database.sessions,
      this.database.items,
      async () => {
        await this.database.items
          .where('sessionId')
          .equals(sessionId)
          .delete()
        await this.database.sessions.delete(sessionId)
      },
    )
  }
}
