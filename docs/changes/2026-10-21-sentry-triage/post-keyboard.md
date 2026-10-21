# Dismiss the post composer keyboard on submission

Source: https://userinput.app/d/did:plc:eyt6ogw5ux2vmp2cn7qpz4up/3myf37bosei2j

Source record: `at://did:plc:eyt6ogw5ux2vmp2cn7qpz4up/app.userinput.discussion/3myf37bosei2j`.

The report describes a successful post returning to the feed while the keyboard
remains open and continues editing the old composer text. Widget regression
tests reproduced the missing keyboard dismissal immediately after pressing
Post, before asynchronous publication completes.

After local validation succeeds, the standard post composer now unfocuses its
focus scope before asynchronous preparation, confirmation, or publication.
Empty/invalid text retains editor focus for correction. Publication failure
preserves the text and allows the user to focus the editor again. The shared
submission entry point covers ordinary posts, replies, quotes, and scheduled
posts without changing publication or provider lifetimes.

Validation:

- Both new success/failure keyboard cases failed against the unchanged composer
  because the keyboard remained visible after pressing Post.
- Both pass with the fix, checking dismissal while the write is pending,
  continued dismissal after completion, no remaining text-input client after
  successful route removal, and editable retained text on failure.
- Blank-post validation keeps the keyboard open.
- Related facet, discard/alt-text, language, video, scheduling, and composer-entry
  suites: 24 tests passed.
- `fvm flutter analyze`: no issues; `git diff --check`: passed.
- Physical-device keyboard behavior has not been verified. No release or public
  feedback status update was performed.
