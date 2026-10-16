"""Render the Follow-action studies with the Rive CLI. Requires Pillow."""
from pathlib import Path
import subprocess
import sys
from PIL import Image,ImageDraw,ImageFont,ImageChops

ROOT=Path(__file__).resolve().parent
shots=ROOT/'build/frames'
shots.mkdir(parents=True,exist_ok=True)
(ROOT/'previews').mkdir(exist_ok=True)
for i in ([] if '--reuse-frames' in sys.argv else range(40)):
    subprocess.run(['rive',str(ROOT),f'--screenshot={shots / f"frame-{i}.png"}',f'--advance={i*3+1}'],check=True,capture_output=True)
fonts=Path('/System/Library/Fonts/Supplemental')
regular=str(fonts/'Arial.ttf') if (fonts/'Arial.ttf').exists() else 'DejaVuSans.ttf'
bold=str(fonts/'Arial Bold.ttf') if (fonts/'Arial Bold.ttf').exists() else 'DejaVuSans-Bold.ttf'
frames=[]
for i in range(40):
    im=Image.new('RGB',(840,310),'#F5EFE4')
    im.paste(Image.open(shots/f'frame-{i}.png').convert('RGB'),(0,44))
    d=ImageDraw.Draw(im)
    d.text((28,19),'CRAFTSKY  /  THREE WAYS TO TIE A KNOT',font=ImageFont.truetype(bold,15),fill='#1535D6')
    for j,(title,sub) in enumerate([
        ('01  Tie the knot','The original bow'),
        ('02  Gentle pull','Wide loops softly cinch together'),
        ('03  Loop and tuck','One loop rises, the other wraps'),
    ]):
        cx=140+j*280
        if i==0:
            d.text((cx,120),'Follow',anchor='mt',font=ImageFont.truetype(bold,15),fill='#1535D6')
        elif i>=14:
            d.text((cx,158),'Following',anchor='mt',font=ImageFont.truetype(bold,12),fill='#1535D6')
        d.text((cx,231),title,anchor='mt',font=ImageFont.truetype(bold,18),fill='#161210')
        d.text((cx,264),sub,anchor='mt',font=ImageFont.truetype(regular,13),fill='#7A716B')
    frames.append(im)
frames[18].save(ROOT/'previews/comparison.png')
frames[0].save(ROOT/'previews/comparison.gif',save_all=True,append_images=frames[1:],duration=50,loop=0,disposal=2,optimize=True)
a=Image.open(shots/'frame-4.png').convert('RGB');b=Image.open(shots/'frame-11.png').convert('RGB')
for j,name in enumerate(['Tie the knot','Gentle pull','Loop and tuck']):
    box=(j*280,0,(j+1)*280,168)
    assert ImageChops.difference(a.crop(box),b.crop(box)).getbbox(),name+' did not animate'
    print(name+': rendered motion confirmed')
a=Image.open(shots/'frame-25.png').convert('RGB');b=Image.open(shots/'frame-39.png').convert('RGB')
assert ImageChops.difference(a,b).getbbox() is None,'The final connected poses should hold still'
print('All final connected poses settle and hold.')
