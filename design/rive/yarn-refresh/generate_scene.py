"""Generate the drag-controlled yarn refresh indicator."""
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

def curved_thread(parent, start, control1, control2, finish, over=False):
    shape=el('Shape',parent)
    points=el('PointsPath',shape)
    for point,handle,incoming in [(start,control1,False),(finish,control2,True)]:
        x,y=point
        angle=math.atan2(handle[1]-y,handle[0]-x)
        distance=math.hypot(handle[0]-x,handle[1]-y)
        el('CubicDetachedVertex',points,x=x,y=y,
           inRotation=angle if incoming else 0,inDistance=distance if incoming else 0,
           outRotation=0 if incoming else angle,outDistance=0 if incoming else distance)
    # Paint the pale channel first, then the thread above it.
    if over:stroke(shape,SKY,4.1)
    stroke(shape,BLUE,1.7)
    return shape

def bind(node,prop,name):
    el('DataBindContext',node,sourcePathIds=f'1:1-1:{name}',propertyKey=prop)
root=el('Rive',version='1',kind='fragment')
ab=el('Artboard',root,name='YarnRefresh',id='2:1',width=80,height=240,styleId='2:2',defaultStateMachineId='2:3',viewModelId='1:1',viewModelInstanceId='1:2')
el('LayoutComponentStyle',ab,id='2:2')
check=el('Node',ab,x=40,y=30,opacity=0)
bind(check,18,8)
path(check,[(-7,0),(-2,5),(9,-7)],BLUE,3)
yarn=el('Node',ab,x=40,y=22)
bind(yarn,18,7)
ball=el('Node',yarn,name='Unwinding ball')
bind(ball,15,5);bind(ball,16,4);bind(ball,17,4)
# Three lifted diagonal wraps cross the curved windings behind them.
for points in [
    ((-13,-9),(-9,-1),(0,10),(8,15)),
    ((-9,-14),(-5,-5),(5,6),(13,10)),
    ((-3,-17),(1,-9),(11,0),(16,4)),
]:curved_thread(ball,*points,over=True)
# Curved courses change direction around the ball instead of parallel stripes.
for points in [
    ((-16,-5),(-13,-13),(0,-19),(11,-13)),
    ((-17,1),(-11,-8),(6,-13),(16,-6)),
    ((-16,6),(-8,-2),(8,-6),(17,0)),
    ((-12,12),(-4,3),(10,0),(16,6)),
    ((-6,16),(1,8),(9,6),(13,11)),
]:curved_thread(ball,*points)
circle(ball,0,0,18,None,BLUE,2.5)
circle(ball,0,0,18,SKY)
strand=el('Shape',yarn,name='Vertical strand')
p=el('PointsPath',strand)
# Vertical tangents keep the loose strand soft as its length follows the drag.
for index,(x,y,prop) in enumerate([(0,0,None),(6,74/3,10),(-6,148/3,11),(0,74,3)]):
    v=el('CubicDetachedVertex',p,x=x,y=y,
         inRotation=-math.pi/2,inDistance=74/6 if index else 0,
         outRotation=math.pi/2,outDistance=74/6 if index<3 else 0)
    if prop:bind(v,25,prop)
    if index:bind(v,85,12)
    if index<3:bind(v,87,12)
stroke(strand,BLUE,2.8)
sm=el('StateMachine',ab,name='YarnRefresh',id='2:3')
layer=el('StateMachineLayer',sm,name='Host-driven pose')
el('StateTransition',el('EntryState',layer),stateToId='2:4')
el('AnimationState',layer,x=220,animationId='2:5',id='2:4')
el('LinearAnimation',ab,name='Pose',id='2:5',fps=60,duration=1,loopValue='oneShot')
for color in ab.iter('SolidColor'):
    if color.get('colorValue')==BLUE:bind(color,37,9)
vm=el('ViewModel',root,name='YarnRefresh',id='1:1',defaultInstanceId='1:2')
props=[(3,'strandLength',74),(4,'ballScale',1),(5,'rotation',0),(7,'yarnOpacity',1),(8,'checkOpacity',0),(10,'strandFirstY',74/3),(11,'strandSecondY',148/3),(12,'strandHandle',74/6)]
for i,name,val in props:el('ViewModelPropertyNumber',vm,name=name,id=f'1:{i}')
el('ViewModelPropertyColor',vm,name='inkColor',id='1:9')
inst=el('ViewModelInstance',vm,name='Default',id='1:2',exports='true')
for i,name,val in props:el('ViewModelInstanceNumber',inst,viewModelPropertyId=f'1:{i}',propertyValue=val)
el('ViewModelInstanceColor',inst,viewModelPropertyId='1:9',propertyValue=BLUE)
E.indent(root)
(ROOT/'scene.rml').write_text(E.tostring(root,encoding='unicode')+'\n')
