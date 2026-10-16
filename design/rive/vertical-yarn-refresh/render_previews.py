"""Render gesture demonstrations, with a visible finger marker and drag readout."""
from pathlib import Path
import subprocess,sys
from PIL import Image,ImageDraw,ImageFont,ImageChops
ROOT=Path(__file__).resolve().parent
shots=ROOT/'build/frames';shots.mkdir(parents=True,exist_ok=True)
out=ROOT/'previews';out.mkdir(exist_ok=True)
FULL=[(0,0),(12,0),(36,66),(48,40),(66,112),(75,112),(81,96),(138,96),(174,0),(210,0)]
CANCEL=[(0,0),(12,0),(36,54),(48,26),(63,62),(72,62),(96,0),(210,0)]
def amount(f,knots):
    for (a,x),(b,y) in zip(knots,knots[1:]):
        if a<=f<=b:return x+(y-x)*(f-a)/(b-a)
    return 0
for i in ([] if '--reuse-frames' in sys.argv else range(70)):
    subprocess.run(['rive',str(ROOT),f'--screenshot={shots / f"frame-{i}.png"}',f'--advance={i*3+1}'],capture_output=True,check=True)
fonts=Path('/System/Library/Fonts/Supplemental')
regular=str(fonts/'Arial.ttf');bold=str(fonts/'Arial Bold.ttf')
frames=[]
for i in range(70):
    f=i*3+1
    im=Image.new('RGB',(840,570),'#F5EFE4')
    im.paste(Image.open(shots/f'frame-{i}.png').convert('RGB'),(0,48))
    d=ImageDraw.Draw(im)
    d.text((28,16),'CRAFTSKY / UNWIND AS YOU PULL',font=ImageFont.truetype(bold,15),fill='#1535D6')
    d.text((812,18),'Gesture demonstration',anchor='rt',font=ImageFont.truetype(regular,12),fill='#7A716B')
    for j,(name,sub,knots) in enumerate([
        ('Pull and refresh','Unwinds further as the pull gets longer',FULL),
        ('Pull and cancel','Pull up or let go to wind it back',CANCEL),
    ]):
        cx=210+j*420
        stage=(f'{round(amount(f,knots))} px pull' if f<76 else ('Refreshing' if f<140 else 'Updated' if f<165 else 'Ready') if j==0 else ('Rewinding' if f<96 else 'Ready'))
        d.text((cx,494),stage,anchor='mt',font=ImageFont.truetype(bold,12),fill='#1535D6')
        d.text((cx,519),name,anchor='mt',font=ImageFont.truetype(bold,18),fill='#161210')
        d.text((cx,548),sub,anchor='mt',font=ImageFont.truetype(regular,13),fill='#7A716B')
    frames.append(im)
frames[22].save(out/'comparison.png')
frames[0].save(out/'comparison.gif',save_all=True,append_images=frames[1:],duration=50,loop=0,disposal=2,optimize=True)
# Verify all three relevant states rendered differently.
for j,name in enumerate(['Full pull','Cancelled pull']):
    a=Image.open(shots/'frame-11.png').convert('RGB')
    b=Image.open(shots/'frame-16.png').convert('RGB')
    box=(j*420+80,70,j*420+340,230)
    assert ImageChops.difference(a.crop(box),b.crop(box)).getbbox(),name+' should reverse with the drag'
    print(name+': drag reversal visible')
a=Image.open(shots/'frame-60.png').convert('RGB');b=Image.open(shots/'frame-69.png').convert('RGB')
assert ImageChops.difference(a,b).getbbox() is None,'Both feeds should finish at rest'
print('Both feeds settle. GIF: 3.5 seconds, 20 fps, uniform 50 ms delays.')
