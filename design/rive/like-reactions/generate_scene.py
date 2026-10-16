"""Generate editable RML for three craft-themed like-reaction studies."""
from pathlib import Path
import math
import xml.etree.ElementTree as E

ROOT = Path(__file__).parent
uid = 10
keys = []

def el(tag, parent=None, **attrs):
    node = E.Element(tag, {k: str(v) for k, v in attrs.items()})
    if parent is not None:
        parent.append(node)
    return node

def ident():
    global uid
    uid += 1
    return f'0:{uid}'

def key(node, prop, values, ease=True):
    if 'id' not in node.attrib:
        node.set('id', ident())
    obj = el('KeyedObject', objectId=node.get('id'))
    kp = el('KeyedProperty', obj, propertyKey=prop)
    for frame, value in values:
        k = el('KeyFrameDouble', kp, frame=frame, value=round(value, 5),
               interpolationType='cubic' if ease else 'linear')
        if ease:
            el('CubicEaseInterpolator', k, x1='.22', y1='0', x2='.3', y2='1')
    keys.append(obj)

def fill(shape, color):
    return el('SolidColor', el('Fill', shape), colorValue=color)

def stroke(shape, color, width=3, dash=False, trim=False):
    st = el('Stroke', shape, thickness=width, cap='round', join='round')
    el('SolidColor', st, colorValue=color)
    if trim:
        tr = el('TrimPath', st, start=0, end=0)
    if dash:
        dp = el('DashPath', st)
        el('Dash', dp, length=3.2)
        el('Dash', dp, length=4.3)
    return tr if trim else st

def circle(parent, x, y, r, color=None, outline=None, width=2):
    s = el('Shape', parent, x=x, y=y)
    el('Ellipse', s, width=r*2, height=r*2, originX='.5', originY='.5')
    if color: fill(s, color)
    if outline: stroke(s, outline, width)
    return s

def rect(parent, x, y, w, h, color, radius=0):
    s = el('Shape', parent, x=x, y=y)
    el('Rectangle', s, width=w, height=h, originX='.5', originY='.5', cornerRadiusTL=radius)
    fill(s, color)
    return s

def line(parent, points, color, width=3):
    s = el('Shape', parent)
    p = el('PointsPath', s)
    for x, y in points:
        el('StraightVertex', p, x=x, y=y)
    stroke(s, color, width)
    return s

# Closed heart made of four cubic segments. Keep independent handles at the
# pointed base and cleft; they should stay crisp when the outline draws on.
ends = [(0,31),(-34,-11),(0,-20),(34,-11)]
controls = [((-12,23),(-34,6)),((-34,-34),(-9,-38)),((9,-38),(34,-34)),((34,6),(12,23))]

def heart(parent, color=None, outline=None, width=3):
    s = el('Shape', parent)
    p = el('PointsPath', s, isClosed='true', isClockwise='true')
    for i, (x,y) in enumerate(ends):
        inc = controls[(i-1)%4][1]
        out = controls[i][0]
        def handle(pt):
            return math.atan2(pt[1]-y,pt[0]-x), math.hypot(pt[0]-x,pt[1]-y)
        a,b=handle(inc); c,d=handle(out)
        el('CubicDetachedVertex',p,x=x,y=y,inRotation=a,inDistance=b,outRotation=c,outDistance=d)
    if color: fill(s,color)
    if outline: stroke(s,outline,width)
    return s

def pop(parent, tilt=0):
    n=el('Node',parent,name='Like reaction')
    key(n,18,[(0,0),(3,1),(56,1),(76,0),(84,0)])
    for prop in [16,17]: key(n,prop,[(0,.35),(12,1.14),(20,.97),(26,1),(56,1),(76,.88),(84,.88)])
    key(n,15,[(0,tilt-.15),(14,tilt+.045),(26,tilt),(84,tilt)])
    key(n,14,[(0,7),(16,0),(56,0),(76,-8),(84,-8)])
    return n

def confetti(parent, squares=False, accent=None):
    for i,a in enumerate([-.95,-2.2,.2,2.9,1.1,2.0]):
        n=el('Node',parent,name=f'Stitch spark {i+1}')
        key(n,13,[(0,math.cos(a)*26),(8,math.cos(a)*26),(29,math.cos(a)*57),(54,math.cos(a)*63),(84,math.cos(a)*63)])
        key(n,14,[(0,math.sin(a)*26),(8,math.sin(a)*26),(29,math.sin(a)*49),(54,math.sin(a)*55),(84,math.sin(a)*55)])
        key(n,18,[(0,0),(8,0),(16,1),(33,1),(50,0),(84,0)])
        key(n,15,[(0,0),(50,.8 if i%2 else -.8),(84,0)])
        color=accent or ['FF1535D6','FFF03A2E','FFF7D46A'][i%3]
        if squares:
            r=rect(n,0,0,6,6,color,.8);r.set('rotation','.65')
            line(n,[(-1,-1),(1,1)],'FFFFFFFF',1)
        else:
            line(n,[(-3,0),(3,0)],'FFFFFFFF',5)
            line(n,[(0,-3),(0,3)],'FFFFFFFF',5)
            line(n,[(-3,0),(3,0)],color,2)
            line(n,[(0,-3),(0,3)],color,2)
            n[:] = list(reversed(list(n)))

def stitched(parent):
    n=pop(parent)
    # First declared is painted on top in RML.
    # Reveal only the actual dashed seam. A separate leading overlay creates
    # a longer solid mark ahead of the stitches, especially near closure.
    h=heart(n)
    tr=stroke(h,'FFF03A2E',3.5,dash=True,trim=True)
    key(tr,115,[(0,0),(8,0),(36,1),(84,1)],False)
    heart(n,'FFFDDED9','FFFFFFFF',8)
    confetti(n,accent='FFF03A2E')

def quilt(parent):
    n=pop(parent,-.055)
    baseid=ident()
    seams=el('Shape',n,name='Running stitch seams')
    el('ClippingShape',seams,sourceId=baseid)
    for pts in [[(0,-36),(0,36)],[(-40,-2),(40,-2)]]:
        p=el('PointsPath',seams)
        for x,y in pts:el('StraightVertex',p,x=x,y=y)
    stroke(seams,'FFFFFFFF',2.1,dash=True)
    for i,(x,y,c) in enumerate([(-21,-21,'FFC9B8E8'),(21,-21,'FFF03A2E'),(-21,19,'FFE27B4A'),(21,19,'FFF7D46A')]):
        patch=el('Node',n,name=f'Fabric patch {i+1}')
        key(patch,13,[(0,x*.4),(8+i*2,x*.4),(22+i*2,0),(84,0)])
        key(patch,14,[(0,y*.4),(8+i*2,y*.4),(22+i*2,0),(84,0)])
        r=rect(patch,x,y,42,40,c)
        el('ClippingShape',r,sourceId=baseid)
        # Tiny woven lines inside each fabric panel.
        for yy in [-10,-3,4,11]:
            l=line(patch,[(x-20,y+yy),(x+20,y+yy)],'20FFFFFF',.7)
            el('ClippingShape',l,sourceId=baseid)
        patch.remove(r); patch.append(r)
    base=heart(n,'FFF03A2E','FFFFFFFF',7);base.set('id',baseid)
    confetti(n,True)

def button(parent):
    n=pop(parent,.08)
    threads=el('Shape',n,name='Sewn cross thread')
    for points in [[(-9,-8),(9,10)],[(9,-8),(-9,10)]]:
        s=line(threads,points,'FFF5EFE4',2.5)
        st=s.find('Stroke');tr=el('TrimPath',st,start=0,end=0)
        offset=0 if points[0][0]<0 else 6
        key(tr,115,[(0,0),(12+offset,0),(28+offset,1),(84,1)])
    for x in [-9,9]:
        for y in [-8,10]:
            circle(n,x,y,3.8,'FFB82016','FFFD8C81',1.3)
    inset=el('Node',n,scaleX='.84',scaleY='.84')
    heart(inset,None,'FFFD8C81',1.8)
    heart(n,'FFF03A2E','FFFFFFFF',7)
    shadow=el('Node',n,y=3)
    heart(shadow,'FFB82016')
    confetti(n)

def backdrop(parent,x,color):
    n=el('Node',parent,x=x,y=112)
    for t in range(-96,97,8):
        line(n,[(t,-84),(t,84)],'28FFFFFF',1)
    for t in range(-80,81,8):
        line(n,[(-108,t),(108,t)],'1BFFFFFF',1)
    rect(n,0,0,232,192,color,12)

def artboard(name,x,y,w,h,functions,comparison=False):
    global keys
    keys=[]
    abid,style,sm,state,anim=[ident() for _ in range(5)]
    ab=el('Artboard',name=name,id=abid,x=x,y=y,width=w,height=h,styleId=style,defaultStateMachineId=sm)
    el('LayoutComponentStyle',ab,id=style)
    if comparison:fill(ab,'FFF5EFE4')
    for cx,cy,fn in functions:
        fn(el('Node',ab,x=cx,y=cy))
    if comparison:
        for cx,c in zip([140,420,700],['FF9BC2E6','FFEFE7D6','FFC9B8E8']):backdrop(ab,cx,c)
    m=el('StateMachine',ab,name='Like',id=sm);layer=el('StateMachineLayer',m,name='Reaction')
    el('StateTransition',el('EntryState',layer),stateToId=state)
    el('AnimationState',layer,x=220,animationId=anim,id=state)
    timeline=el('LinearAnimation',ab,name='Like reaction',id=anim,fps=60,duration=144 if comparison else 84,loopValue='loop' if comparison else 'oneShot')
    timeline.extend(keys)
    return ab

root=el('Rive',version='1',kind='fragment')
root.append(artboard('Comparison',0,0,840,224,[(140,112,stitched),(420,112,quilt),(700,112,button)],True))
for i,(name,fn) in enumerate([('Stitched heart',stitched),('Patchwork pop',quilt),('Button love',button)]):
    root.append(artboard(name,i*200,280,160,160,[(80,80,fn)]))
E.indent(root)
(ROOT/'scene.rml').write_text(E.tostring(root,encoding='unicode')+'\n')
