import { describe, expect, it } from 'vitest'

import {
  classifyPostMetadataPath,
  isSupportedInstagramPostMediaPath,
  resolveArchiveMediaPath,
} from './paths'

describe('supported Instagram export paths (UT-002)', () => {
  it('accepts exact post metadata paths with one optional wrapper', () => {
    expect(
      classifyPostMetadataPath(
        'your_instagram_activity/media/posts_1.json',
      ),
    ).toEqual({ wrapper: '', variant: 'legacy' })
    expect(
      classifyPostMetadataPath(
        'instagram-user-2026/your_instagram_activity/media/posts.json',
      ),
    ).toEqual({
      wrapper: 'instagram-user-2026/',
      variant: 'detailed',
    })
  })

  it('rejects traversal, absolute, confusable, and over-wrapped paths', () => {
    for (const path of [
      '../your_instagram_activity/media/posts.json',
      '/your_instagram_activity/media/posts.json',
      'C:/your_instagram_activity/media/posts.json',
      'a/b/your_instagram_activity/media/posts.json',
      'your_instagram_activity/media/Posts.json',
      'your_instagram_activity/media/posts.txt',
      'your_instagram_activity/messages/posts.json',
    ]) {
      expect(classifyPostMetadataPath(path)).toBeNull()
    }
  })

  it('resolves a source media URI only inside the same archive wrapper', () => {
    expect(
      resolveArchiveMediaPath(
        'instagram-user-2026/',
        'media/posts/photo.jpg',
      ),
    ).toBe('instagram-user-2026/media/posts/photo.jpg')
    expect(() =>
      resolveArchiveMediaPath('', '../messages/private.json'),
    ).toThrow()
  })

  it('allowlists only observed Instagram post-media directories', () => {
    for (const path of [
      'media/posts/photo.jpg',
      'media/posts/202603/photo.jpg',
      'media/other/photo.jpg',
    ]) {
      expect(isSupportedInstagramPostMediaPath(path)).toBe(true)
    }
    for (const path of [
      'messages/private.jpg',
      'media/messages/private.jpg',
      'media/stories/private.jpg',
      'media/posts/nested/private.jpg',
      'media/posts/20260/private.jpg',
      'media/posts/202603/nested/private.jpg',
      'media/other/nested/private.jpg',
      'media/posts/photo.jpg?private=true',
      'media/posts/../messages/private.jpg',
    ]) {
      expect(isSupportedInstagramPostMediaPath(path)).toBe(false)
      expect(() => resolveArchiveMediaPath('', path)).toThrow()
    }
  })
})
