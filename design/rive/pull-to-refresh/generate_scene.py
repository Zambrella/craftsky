"""Generate editable RML for three craft-themed follow-action studies."""
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


BLUE='FF1535D6'
RED='FFF03A2E'
PAPER='FFF5EFE4'
SKY='FFB8D5EB'

def path(parent,points,color,width=3,reveal=None,dash=False,closed=False):
    s=el('Shape',parent)
    p=el('PointsPath',s,isClosed=str(closed).lower())
    for x,y in points:el('StraightVertex',p,x=x,y=y)
    tr=stroke(s,color,width,trim=reveal is not None,dash=dash)
    if reveal:key(tr,115,[(0,0),(reveal[0],0),(reveal[1],1),(216,1)])
    return s

def visible(parent,name):
    n=el('Node',parent,name=name)
    key(n,18,[(0,0),(8,0),(18,1),(126,1),(135,0),(216,0)])
    key(n,17,[(0,.5),(30,1),(216,1)])
    key(n,16,[(0,.5),(30,1),(216,1)])
    return n

def yarn(parent):
    n=visible(parent,'Unspooling yarn')
    # A continuous strand grows with the pull, while the ball winds on release.
    thread=el('Node',n,name='Loose strand')
    key(thread,13,[(0,0),(42,0),(62,2),(83,-2),(104,2),(126,0),(216,0)])
    pts=[(14,0),(19,2),(23,8),(29,8),(33,5)]
    path(thread,pts,RED,2.6,(8,30))
    ball=el('Node',n,name='Yarn winds')
    key(ball,15,[(0,-1),(36,0),(42,0),(126,math.tau*2),(216,math.tau*2)],False)
    for offset in [-8,-2,4,10]:
        pts=[]
        for x in range(-12,13,2):
            y=.55*x+offset
            if x*x+y*y<13*13:pts.append((x,y))
        if len(pts)>1:path(ball,pts,BLUE,1.8)
    circle(ball,0,0,14,None,BLUE,2.7)
    circle(ball,0,0,14,SKY)

def stitches(parent):
    n=visible(parent,'Stitch circle')
    spin=el('Node',n,name='Thread circles')
    key(spin,15,[(0,-math.pi/2),(42,-math.pi/2),(126,math.tau*2-math.pi/2),(216,math.tau*2-math.pi/2)],False)
    ring=el('Shape',spin)
    el('Ellipse',ring,width=30,height=30,originX='.5',originY='.5')
    tr=stroke(ring,RED,2.8,dash=True,trim=True)
    key(tr,115,[(0,0),(30,.82),(126,.82),(216,.82)])
    arrow=el('Node',n,name='Pull arrow')
    key(arrow,18,[(0,1),(34,1),(44,0),(216,0)])
    path(arrow,[(0,-6),(0,6)],BLUE,2.3)
    path(arrow,[(-4,2),(0,6),(4,2)],BLUE,2.3)

def weave(parent):
    n=visible(parent,'Little loom')
    shuttle=el('Node',n,name='Weaving shuttle')
    key(shuttle,13,[(0,-16),(42,-16),(63,16),(84,-16),(105,16),(126,-16),(216,-16)])
    rect(shuttle,0,0,10,5,RED,2.5)
    for i,y in enumerate([-10,0,10]):
        row=el('Node',n,name=f'Woven course {i+1}')
        vals=[(0,0),(8+i*6,0),(20+i*6,1),(42,1)]
        for frame in range(49,127,7):
            vals.append((frame,.35 if ((frame-49)//7-i)%3==0 else 1))
        vals.append((216,1));key(row,18,vals)
        # White gaps give each thread an over-under rhythm.
        for j,x in enumerate([-12,-4,4,12]):
            if (i+j)%2==0:path(row,[(x,y-3),(x,y+3)],PAPER,4)
        path(row,[(-17,y),(17,y)],RED,2.8)
    for x in [-12,-4,4,12]:path(n,[(x,-16),(x,16)],BLUE,2.3)

def completion(parent):
    n=el('Node',parent,name='Refreshed')
    key(n,18,[(0,0),(132,0),(141,1),(155,1),(168,0),(216,0)])
    for prop in [16,17]:key(n,prop,[(0,.8),(132,.8),(144,1.08),(153,1),(216,1)])
    path(n,[(-7,0),(-2,5),(8,-6)],BLUE,3)

def panel(ab,cx,fn):
    n=el('Node',ab,x=cx)
    # Foreground feed hides the indicator naturally as the pull closes.
    feed=el('Node',n,name='Feed follows finger',y=88)
    key(feed,14,[(0,88),(30,166),(40,166),(48,158),(153,158),(184,88),(216,88)])
    # Feed post details and neutral artwork, front-to-back.
    for y,w in [(22,74),(36,44)]:rect(feed,-39+w/2,y,w,4,'FFCEC8BF',2)
    circle(feed,-85,28,12,'FFE7C66E')
    for y,w in [(190,162),(202,112)]:rect(feed,-98+w/2,y,w,5,'FFD8D1C7',2)
    # A quiet abstract fabric illustration.
    for x in [-64,-32,0,32,64]:path(feed,[(x-30,63),(x+35,179)],'44FFFFFF',9)
    for y in [85,112,139,166]:path(feed,[(-102,y),(102,y)],'44FFFFFF',7)
    rect(feed,0,122,208,116,SKY,9)
    rect(feed,0,107,236,214,'FFFFFFFF',12)
    indicator=el('Node',n,x=0,y=123,name='Refresh indicator')
    completion(indicator);fn(indicator)
    # Fixed app header above the reveal gap.
    circle(n,-92,52,7,BLUE)
    rect(n,-33,52,76,6,BLUE,3)
    for xx in [87,96,105]:circle(n,xx,52,1.4,BLUE)
    rect(n,0,52,240,40,'FFFFFFFF',10)
    base=rect(n,0,205,252,346,'FFFFFFFF',22);stroke(base,'261535D6',1.3)

def artboard(name,w,h,items,comparison=False):
    global keys
    keys=[]
    aid,style,sm,state,anim=[ident() for _ in range(5)]
    ab=el('Artboard',name=name,id=aid,x=0 if comparison else (uid%3)*160,y=0 if comparison else 500,width=w,height=h,styleId=style,defaultStateMachineId=sm)
    el('LayoutComponentStyle',ab,id=style)
    if comparison:
        fill(ab,PAPER)
        for cx,fn in items:panel(ab,cx,fn)
    else:
        n=el('Node',ab,x=w/2,y=h/2);completion(n);items[0][1](n)
    machine=el('StateMachine',ab,name='Refresh demo',id=sm)
    layer=el('StateMachineLayer',machine,name='Pull, refresh, settle')
    el('StateTransition',el('EntryState',layer),stateToId=state)
    el('AnimationState',layer,x=220,animationId=anim,id=state)
    timeline=el('LinearAnimation',ab,name='Pull and refresh',id=anim,fps=60,duration=216,loopValue='loop')
    timeline.extend(keys)
    return ab

root=el('Rive',version='1',kind='fragment')
root.append(artboard('Comparison',900,440,[(150,yarn),(450,stitches),(750,weave)],True))
for i,(name,fn) in enumerate([('Unspooling yarn',yarn),('Stitch circle',stitches),('Little loom',weave)]):
    ab=artboard(name,100,80,[(0,fn)])
    ab.set('x',str(i*160))
    root.append(ab)
E.indent(root)
(ROOT/'scene.rml').write_text(E.tostring(root,encoding='unicode')+'\n')
