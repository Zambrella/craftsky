import type { ExportVariant } from '../domain/types'

export interface PostMetadataPath {
  readonly wrapper: string
  readonly variant: ExportVariant
}

const POST_PATH =
  /^(?:(?<wrapper>[^./\\][^/\\]*)\/)?your_instagram_activity\/media\/posts(?<number>_\d+)?\.json$/
const POST_MEDIA_PATH =
  /^media\/(?:posts\/(?:\d{6}\/)?|other\/)[^/?#]+$/

export function classifyPostMetadataPath(
  path: string,
): PostMetadataPath | null {
  if (!isSafeArchivePath(path)) return null
  const match = POST_PATH.exec(path)
  if (!match?.groups) return null
  return {
    wrapper: match.groups.wrapper ? `${match.groups.wrapper}/` : '',
    variant: match.groups.number ? 'legacy' : 'detailed',
  }
}

export function isSafeArchivePath(path: string): boolean {
  if (
    !path ||
    path.startsWith('/') ||
    /^[a-zA-Z]:/u.test(path) ||
    path.includes('\\') ||
    path.includes('\u0000')
  ) {
    return false
  }
  const segments = path.split('/')
  return segments.every(
    (segment) => segment.length > 0 && segment !== '.' && segment !== '..',
  )
}

export function isSupportedInstagramPostMediaPath(path: string): boolean {
  return isSafeArchivePath(path) && POST_MEDIA_PATH.test(path)
}

export function resolveArchiveMediaPath(
  wrapper: string,
  sourcePath: string,
): string {
  if (!isSupportedInstagramPostMediaPath(sourcePath)) {
    throw new Error('unsupportedPostMediaPath')
  }
  if (wrapper && (!wrapper.endsWith('/') || !isSafeArchivePath(wrapper.slice(0, -1)))) {
    throw new Error('unsafeArchiveWrapper')
  }
  return `${wrapper}${sourcePath}`
}
