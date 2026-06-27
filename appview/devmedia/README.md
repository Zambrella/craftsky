# Dev Media Fixtures

`cli seed demo` references these image fixture names in dev-only database rows.

If a file exists here, `/v1/dev/media/{name}` serves it. If no matching file is
present, AppView generates a deterministic textile-like JPEG placeholder so the
demo seed still works out of the box.

Accepted extensions: `.jpg`, `.jpeg`, `.png`, `.webp`.

Current demo seed fixture names:

- `knit-cardigan-moss`
- `linen-dress-rust`
- `quilt-top-navy`
- `crochet-blanket-squares`
- `visible-mending-denim`
- `naturally-dyed-skeins`
- `avatar-viewer`
- `avatar-mara`
- `avatar-jules`
- `avatar-nina`
- `avatar-sol`
- `avatar-bea`
- `avatar-ori`
- `banner-viewer`
- `banner-mara`
- `banner-jules`
- `banner-nina`
- `banner-sol`
- `banner-bea`
- `banner-ori`

The expanded generated dataset also references project image names like
`generated-project-05`, `generated-project-06`, and so on. You can provide any
of those by adding a matching file, or leave them absent to use the deterministic
generated placeholders.
