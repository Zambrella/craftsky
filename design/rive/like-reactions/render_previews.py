"""Render a labelled comparison. Requires the Rive CLI and Pillow."""
from pathlib import Path
import subprocess
import sys
from PIL import Image, ImageDraw, ImageFont, ImageChops

ROOT=Path(__file__).resolve().parent
frames_dir=ROOT/'build/frames'
frames_dir.mkdir(parents=True,exist_ok=True)
(ROOT/'previews').mkdir(exist_ok=True)
for i in ([] if '--reuse-frames' in sys.argv else range(144)):
    target=frames_dir/f'frame-{i}.png'
    subprocess.run(['rive',str(ROOT),f'--screenshot={target}',f'--advance={i+1}'],check=True,capture_output=True)

font_root=Path('/System/Library/Fonts/Supplemental')
regular=str(font_root/'Arial.ttf') if (font_root/'Arial.ttf').exists() else 'DejaVuSans.ttf'
bold=str(font_root/'Arial Bold.ttf') if (font_root/'Arial Bold.ttf').exists() else 'DejaVuSans-Bold.ttf'
frames=[]
for i in range(144):
    im=Image.new('RGB',(840,370),'#F5EFE4')
    capture=Image.open(frames_dir/f'frame-{i}.png').convert('RGB')
    im.paste(capture,(0,54))
    d=ImageDraw.Draw(im)
    d.text((24,22),'CRAFTSKY  /  A LITTLE LOVE, HANDMADE',font=ImageFont.truetype(bold,15),fill='#1535D6')
    for j,(title,subtitle) in enumerate([
        ('01  Stitched heart','A little love, sewn in thread'),
        ('02  Patchwork pop','Little pieces, one lovely heart'),
        ('03  Button love','A heart, fastened with a stitch'),
    ]):
        x=140+j*280
        d.text((x,293),title,anchor='mt',font=ImageFont.truetype(bold,18),fill='#161210')
        d.text((x,325),subtitle,anchor='mt',font=ImageFont.truetype(regular,13),fill='#7A716B')
    frames.append(im)
frames[35].save(ROOT/'previews/comparison.png')
# Capture every Rive frame for inspection, but encode at the original 20 fps.
# Avoid 10 ms GIF delays, which can play inconsistently across viewers.
preview_frames=frames[::3]
frame_durations=50
preview_frames[0].save(ROOT/'previews/comparison.gif',save_all=True,append_images=preview_frames[1:],duration=frame_durations,loop=0,disposal=2,optimize=True)
stitched=[frame.crop((0,54,280,370)) for frame in preview_frames]
stitched[0].save(ROOT/'previews/stitched-heart.gif',save_all=True,append_images=stitched[1:],duration=frame_durations,loop=0,disposal=2,optimize=True)
stitched[0].save(ROOT/'previews/stitched-heart-clean-seam.gif',save_all=True,append_images=stitched[1:],duration=frame_durations,loop=0,disposal=2,optimize=True)
stitched[0].save(ROOT/'previews/stitched-heart-snappy.gif',save_all=True,append_images=stitched[1:],duration=frame_durations,loop=0,disposal=2,optimize=True)

a=Image.open(frames_dir/'frame-12.png').convert('RGB')
b=Image.open(frames_dir/'frame-36.png').convert('RGB')
for j,name in enumerate(['Stitched heart','Patchwork pop','Button love']):
    region=(j*280,0,(j+1)*280,224)
    assert ImageChops.difference(a.crop(region),b.crop(region)).getbbox(),name+' has no rendered motion'
    print(name+': rendered motion confirmed')
# All overlay content must disappear completely during the replay pause.
a=Image.open(frames_dir/'frame-84.png').convert('RGB')
b=Image.open(frames_dir/'frame-143.png').convert('RGB')
assert ImageChops.difference(a,b).getbbox() is None,'Unexpected motion/content after the one-shot finishes'
print('Clean, static replay pause confirmed')
