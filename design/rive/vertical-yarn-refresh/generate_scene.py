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

DURATION=210
FULL=[(0,0),(12,0),(36,66),(48,40),(66,112),(75,112),(81,96),(138,96),(174,0),(210,0)]
CANCEL=[(0,0),(12,0),(36,54),(48,26),(63,62),(72,62),(96,0),(210,0)]

def amount(frame,cancel=False):
    knots=CANCEL if cancel else FULL
    for (a,x),(b,y) in zip(knots,knots[1:]):
        if a<=frame<=b:return x+(y-x)*(frame-a)/(b-a)
    return 0

def motion(node,prop,fn,cancel=False):
    key(node,prop,[(f,fn(amount(f,cancel),f)) for f in range(0,DURATION+1,3)],False)

def yarn(parent,cancel=False):
    n=el('Node',parent,name='Drag-coupled yarn',y=88)
    if cancel:motion(n,18,lambda d,f:min(1,d/20),cancel)
    # The ball stays beneath the header; its red strand ends at the feed edge.
    ball=el('Node',n,name='Unwinding ball')
    def turn(d,f):
        if cancel or f<=75:return d/112*math.pi*2
        if f<=138:return math.pi*2+max(0,f-81)/57*math.pi*2
        return d/96*math.pi*4
    motion(ball,15,turn,cancel)
    for prop in [16,17]:
        motion(ball,prop,lambda d,f:math.sqrt(18**2-min(d/112,1)*(18**2-10**2))/18,cancel)
    for off in [-12,-6,0,6,12]:
        pts=[]
        for x in range(-16,17,2):
            y=.65*x+off
            if x*x+y*y<17*17:pts.append((x,y))
        if len(pts)>1:path(ball,pts,BLUE,2)
    circle(ball,0,0,18,None,BLUE,2.5)
    circle(ball,0,0,18,SKY)
    strand=el('Shape',n,name='Vertical yarn reaches the feed')
    p=el('PointsPath',strand)
    el('StraightVertex',p,x=0,y=0)
    end=el('StraightVertex',p,x=0,y=0)
    motion(end,25,lambda d,f:d,cancel)
    stroke(strand,RED,2.8)
    # Success fades the yarn, then the feed closes without an abrupt pop.
    if not cancel:
        key(n,18,[(0,0),(12,0),(18,1),(138,1),(148,0),(210,0)])
        done=el('Node',parent,y=116,name='Refreshed')
        key(done,18,[(0,0),(141,0),(150,1),(162,1),(174,0),(210,0)])
        path(done,[(-6,0),(-1,5),(9,-6)],BLUE,3)

def panel(ab,cx,cancel=False):
    n=el('Node',ab,x=cx)
    # A marker stands for the finger, not a loading indicator.
    finger=el('Node',n,x=77,name='Finger position')
    motion(finger,14,lambda d,f:108+d,cancel)
    key(finger,18,[(0,0),(12,0),(18,.65),(72,.65),(78,0),(210,0)])
    circle(finger,0,0,8,None,RED,1.5)
    circle(finger,0,0,2,RED)
    feed=el('Node',n,name='Feed tracks drag')
    motion(feed,14,lambda d,f:88+d,cancel)
    for y,w in [(22,76),(36,48)]:rect(feed,-39+w/2,y,w,4,'FFCEC8BF',2)
    circle(feed,-85,28,12,'FFE7C66E')
    for y,w in [(190,162),(202,112)]:rect(feed,-98+w/2,y,w,5,'FFD8D1C7',2)
    for x in [-64,-32,0,32,64]:path(feed,[(x-30,63),(x+35,179)],'44FFFFFF',9)
    for y in [85,112,139,166]:path(feed,[(-102,y),(102,y)],'44FFFFFF',7)
    rect(feed,0,122,208,116,SKY,9)
    rect(feed,0,107,236,214,'FFFFFFFF',12)
    yarn(n,cancel)
    circle(n,-92,52,7,BLUE)
    rect(n,-33,52,76,6,BLUE,3)
    for xx in [87,96,105]:circle(n,xx,52,1.4,BLUE)
    rect(n,0,52,240,40,'FFFFFFFF',10)
    base=rect(n,0,229,252,394,'FFFFFFFF',22)
    stroke(base,'261535D6',1.3)

def board(name,x,cancel=None):
    global keys
    keys=[]
    aid,style,sm,state,anim=[ident() for _ in range(5)]
    ab=el('Artboard',name=name,id=aid,x=x,y=0 if cancel is None else 520,width=840 if cancel is None else 300,height=470,styleId=style,defaultStateMachineId=sm)
    el('LayoutComponentStyle',ab,id=style)
    fill(ab,PAPER)
    if cancel is None:
        panel(ab,210);panel(ab,630,True)
    else:panel(ab,150,cancel)
    machine=el('StateMachine',ab,name='Vertical pull demo',id=sm)
    layer=el('StateMachineLayer',machine,name='Gesture demonstration')
    el('StateTransition',el('EntryState',layer),stateToId=state)
    el('AnimationState',layer,x=220,animationId=anim,id=state)
    anim=el('LinearAnimation',ab,name='Pull, reverse, release',id=anim,fps=60,duration=DURATION,loopValue='loop')
    anim.extend(keys)
    return ab

root=el('Rive',version='1',kind='fragment')
root.append(board('Comparison',0))
root.append(board('Pull and refresh',0,False))
root.append(board('Pull and cancel',360,True))
E.indent(root)
(ROOT/'scene.rml').write_text(E.tostring(root,encoding='unicode')+'\n')
# The motion is sampled from a single drag-distance function, not independent
# easing curves: feed offset, strand endpoint and unwinding remain in lockstep.
for cancel in [False,True]:
    for frame in range(211):
        d=amount(frame,cancel)
        assert 0<=d<=112
print('Both gesture demonstrations use one shared vertical drag value.')
