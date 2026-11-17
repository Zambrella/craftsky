import json
from pathlib import Path
ROOT=Path(__file__).parent
concepts=[
 ('A','needle-plus','Needle plus','Your idea: a sewing needle crosses a knitting needle to form a true plus.', '''<path d="M3.5 10.85H19.8L22 12L19.8 13.15H3.5Z" fill="currentColor"/><circle cx="3" cy="12" r="2" fill="currentColor"/><path fill-rule="evenodd" d="M12 1.5C10.7 1.5 10 2.6 10 4.5L10.8 16L12 22.5L13.2 16L14 4.5C14 2.6 13.3 1.5 12 1.5ZM12 3.1C11.55 3.1 11.3 3.6 11.3 4.5V6.2C11.3 7.4 12.7 7.4 12.7 6.2V4.5C12.7 3.6 12.45 3.1 12 3.1Z" fill="currentColor"/>'''),
 ('B','crossed-tools','Crossed tools','A more literal pair of tools, with a small plus to make the tier explicit.', '''<path d="M4 19L15.5 7.5" stroke="currentColor" stroke-width="2" stroke-linecap="round"/><circle cx="4" cy="19" r="1.8" fill="currentColor"/><path fill-rule="evenodd" d="M3.5 3.5C2.6 4.4 3.1 5.6 4.3 6.8L19.8 20.2L6.8 4.3C5.6 3.1 4.4 2.6 3.5 3.5ZM4.2 4.2L3.9 4.7L5.7 6.5L6.5 5.7L4.7 3.9Z" fill="currentColor"/><path d="M19 2V9M15.5 5.5H22.5" stroke="currentColor" stroke-width="2.1" stroke-linecap="round"/>'''),
 ('C','stitch-plus','Stitch plus','Five small stitches make a plus. The simplest craft-inspired badge.', '''<path d="M12 3.5V8M12 16V20.5M3.5 12H8M16 12H20.5" stroke="currentColor" stroke-width="2.7" stroke-linecap="round"/><circle cx="12" cy="12" r="1.6" fill="currentColor"/>'''),
 ('D','button-plus','Button plus','A button-like ring around a clear plus. Strong at very small sizes.', '''<circle cx="12" cy="12" r="9" stroke="currentColor" stroke-width="2" fill="none"/><path d="M12 7.5V16.5M7.5 12H16.5" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>'''),
 ('E','spool-plus','Spool plus','A compact thread spool carries the plus in its centre.', '''<path d="M5 3H19V6H17V18H19V21H5V18H7V6H5Z" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round" fill="none"/><path d="M12 8.5V15.5M9 12H15" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/>'''),
 ('F','fabric-label','Fabric label','A folded fabric label with a plus. Works across different crafts.', '''<path d="M5 3H15L20 8V21H5Z" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round" fill="none"/><path d="M15 3V8H20M12.5 10V17M9 13.5H16" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" fill="none"/>'''),
]
data=[]
for letter,slug,title,desc,body in concepts:
 svg=f'<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none">{body}</svg>'
 (ROOT/f'{slug}.svg').write_text(svg+'\n')
 data.append(dict(letter=letter,slug=slug,title=title,description=desc,svg=svg))
(ROOT/'concepts.json').write_text(json.dumps(data,indent=2)+'\n')
