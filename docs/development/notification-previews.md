# Notification text previews

`GET /v1/notifications` adds optional `preview` and `quotedPreview` objects:

```json
{"text":"Lovely stitches!","projectTitle":"Winter scarf","role":"comment"}
```

`role` is `post`, `comment`, or `reply`. `projectTitle` is omitted when there is
no project title. The preview contains no media. Source/subject/root references
keep their existing routing meanings.

- Replies preview the new comment or reply; `subjectPost` still describes its
  parent for action wording and navigation. A visible parent/root project's title
  accompanies the new reply text.
- Mentions and quotes preview the source content. Quotes additionally carry a
  `quotedPreview` of the visible quoted content, including its project title and
  role, so the row can distinguish quoted posts, comments, and replies.
- Likes and reposts preview the affected content and its project title.

The AppView batches public preview reads after selecting the notification page.
The shared reader checks current owner state, blocks, mutes, moderation and blocks
between a source author and its reply/quote/mention references. Unavailable quote
context is omitted.

Push notifications for replies, mentions and quotes use the same reader immediately
before provider delivery. Visible bodies include a bounded project title and text
excerpt. Text never enters routing data. Title and excerpt budgets preserve both
within the provider's display-text limit; whitespace/control characters are
normalized. Follow, moderation and Instagram-match copy is unchanged.
