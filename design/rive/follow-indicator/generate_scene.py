"""Generate the production original Tie the knot Follow animation."""
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
root.append(artboard('Follow',0,0,160,120,[(80,60,knot)]))
E.indent(root)
(ROOT/'scene.rml').write_text(E.tostring(root,encoding='unicode')+'\n')
