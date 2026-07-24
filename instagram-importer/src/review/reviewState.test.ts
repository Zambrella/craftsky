import { describe, expect, it } from 'vitest'

import type { ReviewPost } from '../domain/types'
import {
  calculateReviewCounts,
  setAllPostSelection,
  setPostSelectionByKey,
  updateCaption,
  updatePostSelection,
  updateMediaSelection,
} from './reviewState'

const post = (overrides: Partial<ReviewPost> = {}): ReviewPost => ({
  itemKey: 'item',
  rkey: '3kexampletid',
  createdAt: '2024-01-02T03:04:05.000Z',
  caption: 'Caption',
  initialCaption: 'Caption',
  media: [
    {
      token: 'media-1',
      kind: 'image',
      mime: 'image/jpeg',
      width: 100,
      height: 100,
      selected: true,
    },
  ],
  warnings: [],
  selected: true,
  needsTextOnlyConfirmation: false,
  textOnlyConfirmed: false,
  ...overrides,
})

describe('review state (UT-016, UT-019)', () => {
  it('recomputes exact counts after per-post and per-media selection', () => {
    const warned = post({
      itemKey: 'warned',
      warnings: ['captionRepaired'],
    })
    const transformed = post({
      itemKey: 'transformed',
      caption: 'changed',
      warnings: ['captionTruncated'],
    })
    const initial = [warned, transformed]
    expect(calculateReviewCounts(initial, 2)).toEqual({
      selectedPosts: 2,
      selectedImages: 2,
      transformedPosts: 2,
      warningPosts: 2,
      skippedPosts: 2,
    })

    const deselected = updatePostSelection(initial, 'warned', false)
    expect(calculateReviewCounts(deselected, 2)).toMatchObject({
      selectedPosts: 1,
      selectedImages: 1,
    })
    const withoutMedia = updateMediaSelection(
      deselected,
      'transformed',
      'media-1',
      false,
    )
    expect(calculateReviewCounts(withoutMedia, 2)).toMatchObject({
      selectedPosts: 1,
      selectedImages: 0,
    })
  })

  it('revalidates caption edits and requires explicit text-only confirmation', () => {
    const edited = updateMediaSelection([post()], 'item', 'media-1', false)
    expect(edited[0]).toMatchObject({
      needsTextOnlyConfirmation: true,
      textOnlyConfirmed: false,
    })

    const overlong = updateCaption(
      edited,
      'item',
      '🧶'.repeat(2_001),
    )
    expect(overlong[0]?.caption).toBe('🧶'.repeat(2_000))
    expect(overlong[0]?.warnings).toContain('captionTruncated')
  })

  it('cannot reselect an empty post through individual or bulk controls', () => {
    const empty = post({
      caption: '',
      initialCaption: '',
      media: [],
      selected: false,
    })

    expect(updatePostSelection([empty], 'item', true)[0]?.selected).toBe(
      false,
    )
    expect(setAllPostSelection([empty], true)[0]?.selected).toBe(false)
    expect(
      setPostSelectionByKey([empty], new Set(['item']), true)[0]
        ?.selected,
    ).toBe(false)
  })
})
