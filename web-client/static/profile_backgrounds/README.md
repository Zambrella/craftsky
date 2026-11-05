# Profile background textures

These six transparent tiled PNGs were downloaded on 2026-08-10 from
<https://ribo.zone/free/graphics/backgrounds>:

- `bayerdark.png`
- `cubedark.png`
- `dotcrossdark.png`
- `scallopdark.png`
- `skewdark.png`
- `x2.png`

The source page says the images were made by the site author, are free to use,
and that edits and recolours are encouraged. It does not require attribution.
CraftSky stores the files locally and applies a profile-colour tint at render
time; it never hotlinks them.

The five `craft-*.svg` tiles are composed from the matching bundled SVG in
`static/crafts/`. Each tile repeats 20px craft icons on the Flutter header's
40px grid, offset by 20px on alternate rows. The profile header uses these and
the PNG patterns as alpha masks with the palette's texture tint at 18% opacity.
