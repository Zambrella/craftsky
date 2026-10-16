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


def curve(parent, segments, color, width=3, reveal=None, backing=False):
    """Cubic path, with an optional reveal applied directly to the geometry."""
    s=el('Shape',parent)
    p=el('PointsPath',s)
    points=[segments[0][0]]+[seg[3] for seg in segments]
    for i,(x,y) in enumerate(points):
        inc=segments[i-1][2] if i else (x,y)
        out=segments[i][1] if i<len(segments) else (x,y)
        def handle(pt):return math.atan2(pt[1]-y,pt[0]-x),math.hypot(pt[0]-x,pt[1]-y)
        a,b=handle(inc);c,d=handle(out)
        el('CubicDetachedVertex',p,x=x,y=y,inRotation=a,inDistance=b,outRotation=c,outDistance=d)
    for paint,w in ([('FFFFFFFF',width+3),(color,width)] if backing else [(color,width)]):
        tr=stroke(s,paint,w,trim=reveal is not None)
        if reveal:
            start,end=reveal
            key(tr,115,[(0,0),(start,0),(end,1),(72,1)])
    return s

def enter(parent):
    n=el('Node',parent,name='Connection')
    key(n,18,[(0,0),(4,1),(72,1)])
    for prop in [16,17]:key(n,prop,[(0,.82),(14,1.04),(24,1),(72,1)])
    return n

def knot(parent):
    n=enter(parent)
    # The knot covers the crossing of the two thread loops.
    band=el('Node',n,name='Knot tightens')
    key(band,18,[(0,0),(25,0),(29,1),(72,1)])
    for prop in [16,17]:key(band,prop,[(0,.25),(26,.25),(33,1.15),(40,1),(72,1)])
    r=rect(band,0,0,12,10,'FF1535D6',3);r.set('rotation','-.3')
    for side,color in [(-1,'FF1535D6'),(1,'FFF03A2E')]:
        loop=el('Node',n,name='Blue thread' if side<0 else 'Red thread')
        key(loop,13,[(0,side*17),(10,side*17),(30,side*-1),(40,0),(72,0)])
        def xy(x,y):return (side*x,y)
        segments=[
            (xy(0,0),xy(9,-8),xy(26,-30),xy(29,-15)),
            (xy(29,-15),xy(33,2),xy(14,7),xy(0,0)),
        ]
        curve(loop,segments,color,3.7,(4,29),True)
        tail=[(xy(2,2),xy(8,8),xy(15,24),xy(25,21))]
        curve(loop,tail,color,3.3,(18,36),True)

def gentle_pull(parent):
    n=enter(parent)
    # Two broad loops slide together and cinch into a soft, symmetric bow.
    band=el('Node',n,name='Soft centre knot')
    key(band,18,[(0,0),(24,0),(30,1),(72,1)])
    for prop in [16,17]:key(band,prop,[(0,.3),(24,.3),(34,1.18),(44,1),(72,1)])
    r=rect(band,0,0,10,12,'FFF03A2E',4);r.set('rotation','.22')
    for side,color in [(-1,'FF1535D6'),(1,'FFF03A2E')]:
        loop=el('Node',n,name='Loop pulls inward')
        key(loop,13,[(0,side*24),(12,side*21),(34,side*-2),(44,0),(72,0)])
        key(loop,15,[(0,side*.24),(26,side*-.08),(44,0),(72,0)])
        key(loop,16,[(0,1.12),(20,1.12),(35,.92),(44,1),(72,1)])
        def xy(x,y):return side*x,y
        curve(loop,[(xy(0,0),xy(13,-21),xy(36,-17),xy(28,-3)),
                    (xy(28,-3),xy(25,7),xy(9,7),xy(0,0))],color,3.7,(3,24),True)
        curve(loop,[(xy(1,4),xy(9,12),xy(6,25),xy(18,25))],color,3.3,(20,38),True)

def loop_tuck(parent):
    n=enter(parent)
    # A red wrap draws around an upright blue loop, then tucks into a bow.
    centre=el('Node',n,name='Tucked centre')
    key(centre,18,[(0,0),(32,0),(37,1),(72,1)])
    for prop in [16,17]:key(centre,prop,[(0,.5),(33,.5),(41,1.1),(48,1),(72,1)])
    r=rect(centre,0,0,11,9,'FFF03A2E',3);r.set('rotation','-.35')
    blue=el('Node',n,name='Blue loop rises')
    key(blue,15,[(0,-.8),(20,.08),(36,0),(72,0)])
    key(blue,14,[(0,10),(20,-2),(36,0),(72,0)])
    curve(blue,[((0,0),(-13,-7),(-13,-35),(-25,-24)),
                ((-25,-24),(-39,-11),(-15,2),(0,0))],'FF1535D6',3.7,(3,22),True)
    curve(blue,[((-2,3),(-12,13),(-28,17),(-28,24))],'FF1535D6',3.3,(13,30),True)
    red=el('Node',n,name='Red loop wraps and tucks')
    key(red,15,[(0,.5),(24,-.1),(42,0),(72,0)])
    key(red,13,[(0,14),(18,14),(34,-2),(42,0),(72,0)])
    curve(red,[((0,0),(12,3),(35,1),(28,-13)),
               ((28,-13),(21,-30),(7,-11),(0,0))],'FFF03A2E',3.7,(14,35),True)
    curve(red,[((1,3),(17,5),(13,23),(23,20))],'FFF03A2E',3.3,(28,44),True)

def artboard(name,x,y,w,h,items,comparison=False):
    global keys
    keys=[]
    abid,style,sm,state,anim=[ident() for _ in range(5)]
    ab=el('Artboard',name=name,id=abid,x=x,y=y,width=w,height=h,styleId=style,defaultStateMachineId=sm)
    el('LayoutComponentStyle',ab,id=style)
    if comparison:fill(ab,'FFF5EFE4')
    for cx,cy,fn in items:
        fn(el('Node',ab,x=cx,y=cy))
    if comparison:
        for cx in [140,420,700]:
            button=rect(ab,cx,82,224,100,'FFFFFFFF',18)
            stroke(button,'261535D6',1.5)
    m=el('StateMachine',ab,name='Follow',id=sm);layer=el('StateMachineLayer',m,name='Connection')
    el('StateTransition',el('EntryState',layer),stateToId=state)
    el('AnimationState',layer,x=220,animationId=anim,id=state)
    timeline=el('LinearAnimation',ab,name='Follow connection',id=anim,fps=60,duration=120 if comparison else 72,loopValue='loop' if comparison else 'oneShot')
    timeline.extend(keys)
    return ab

root=el('Rive',version='1',kind='fragment')
root.append(artboard('Comparison',0,0,840,168,[(140,82,knot),(420,82,gentle_pull),(700,82,loop_tuck)],True))
for i,(name,fn) in enumerate([('Tie the knot',knot),('Gentle pull',gentle_pull),('Loop and tuck',loop_tuck)]):
    root.append(artboard(name,i*200,240,160,120,[(80,60,fn)]))
E.indent(root)
(ROOT/'scene.rml').write_text(E.tostring(root,encoding='unicode')+'\n')
