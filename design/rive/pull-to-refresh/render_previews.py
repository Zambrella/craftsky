"""Render the Rive studies at 20 fps with uniform 50 ms GIF delays."""
from pathlib import Path
import subprocess
import sys
from PIL import Image, ImageDraw, ImageFont, ImageChops

ROOT=Path(__file__).resolve().parent
shots=ROOT/'build/frames'; shots.mkdir(parents=True,exist_ok=True)
out=ROOT/'previews';out.mkdir(exist_ok=True)
for i in ([] if '--reuse-frames' in sys.argv else range(72)):
    subprocess.run(['rive',str(ROOT),f'--screenshot={shots / f"frame-{i}.png"}',f'--advance={i*3+1}'],check=True,capture_output=True)
fonts=Path('/System/Library/Fonts/Supplemental')
regular=str(fonts/'Arial.ttf') if (fonts/'Arial.ttf').exists() else 'DejaVuSans.ttf'
bold=str(fonts/'Arial Bold.ttf') if (fonts/'Arial Bold.ttf').exists() else 'DejaVuSans-Bold.ttf'
frames=[]
for i in range(72):
    im=Image.new('RGB',(900,510),'#F5EFE4')
    im.paste(Image.open(shots/f'frame-{i}.png').convert('RGB'),(0,40))
    d=ImageDraw.Draw(im)
    d.text((25,16),'CRAFTSKY  /  PULL TO REFRESH',font=ImageFont.truetype(bold,15),fill='#1535D6')
    phase=('Pull down' if i<10 else 'Release' if i<14 else 'Refreshing' if i<43 else 'Up to date' if i<53 else 'Settling' if i<62 else 'Ready')
    d.text((875,17),phase,anchor='rt',font=ImageFont.truetype(bold,13),fill='#7A716B')
    for j,(title,sub) in enumerate([
        ('01  Unspooling yarn','Pull out a strand; wind it back in'),
        ('02  Stitch circle','A familiar spinner, stitched in thread'),
        ('03  Little loom','A tiny weave while the feed refreshes'),
    ]):
        cx=150+j*300
        d.text((cx,444),title,anchor='mt',font=ImageFont.truetype(bold,18),fill='#161210')
        d.text((cx,475),sub,anchor='mt',font=ImageFont.truetype(regular,13),fill='#7A716B')
    frames.append(im)
frames[25].save(out/'comparison.png')
frames[0].save(out/'comparison.gif',save_all=True,append_images=frames[1:],duration=50,loop=0,disposal=2,optimize=True)
a=Image.open(shots/'frame-21.png').convert('RGB');b=Image.open(shots/'frame-27.png').convert('RGB')
for j,name in enumerate(['Unspooling yarn','Stitch circle','Little loom']):
    box=(j*300+100,95,j*300+200,150)
    assert ImageChops.difference(a.crop(box),b.crop(box)).getbbox(),name+' did not animate during refresh'
    print(name+': refreshing motion confirmed')
a=Image.open(shots/'frame-64.png').convert('RGB');b=Image.open(shots/'frame-71.png').convert('RGB')
assert ImageChops.difference(a,b).getbbox() is None,'Feed should settle at rest'
print('All feeds return to rest; GIF timing is 3.6 seconds at 20 fps.')
