import { useId, useState, type FormEvent } from 'react'
import { label, type Review } from '../cases/model'

export type Field={key:string;label:string;type?:'text'|'textarea'|'date'|'select'|'checkbox'|'choices'|'operator';required?:boolean;options?:{value:string;label:string}[];default?:string;hint?:string;maxLength?:number}
export type Action={title:string;heading:string;path:string;help:string;fields:Field[];body?:Record<string,unknown>;expectedRevision?:number|null;prepare?:(body:Record<string,unknown>)=>Record<string,unknown>}
export function options(values:string[]){return values.map(value=>({value,label:label(value)}))}
function localNow(){const now=new Date();return new Date(now.getTime()-now.getTimezoneOffset()*60000).toISOString().slice(0,16)}
export function ActionForm({action,review}:{action:Action;review:(r:Review)=>void}){
 const prefix=useId()
 const [values,setValues]=useState<Record<string,string|boolean|string[]>>(()=>Object.fromEntries(action.fields.map(f=>[f.key,f.type==='checkbox'?false:f.type==='choices'?[]:f.default==='now'?localNow():f.default??(f.type==='select'?f.options?.[0]?.value??'':'')])))
 const [error,setError]=useState('')
 function submit(event:FormEvent){
  event.preventDefault();setError('');const body:Record<string,unknown>={...action.body},summary:string[]=[action.help]
  for(const field of action.fields){
   const value=values[field.key]
   if(field.required&&(typeof value==='string'?!value.trim():Array.isArray(value)?!value.length:!value)){setError('Complete the required fields before reviewing this action.');return}
   if(typeof value==='string'&&!value.trim())continue
   body[field.key]=field.type==='date'?new Date(String(value)).toISOString():typeof value==='string'?value.trim():value
   const display=typeof value==='boolean'?(value?'Yes':'No'):Array.isArray(value)?value.map(v=>field.options?.find(o=>o.value===v)?.label??v).join(', '):field.options?.find(o=>o.value===value)?.label??value
   if(display!=='')summary.push(`${field.label}: ${display}`)
  }
  review({title:action.title,endpoint:'',path:action.path,expectedRevision:action.expectedRevision,body:action.prepare?action.prepare(body):body,summary})
 }
 return <details className="action-disclosure"><summary>{action.heading}</summary><form className="case-form" aria-label={action.title} onSubmit={submit}><p>{action.help}</p>{action.fields.map(field=>{
  const change=(value:string|boolean|string[])=>setValues(prior=>({...prior,[field.key]:value}))
  const value=values[field.key]
  return field.type==='choices'?<fieldset className="effect-choices" key={field.key}><legend>{field.label}</legend>{field.options?.map(option=><label key={option.value}><input type="checkbox" checked={(value as string[]).includes(option.value)} onChange={e=>change(e.target.checked?[...(value as string[]),option.value]:(value as string[]).filter(v=>v!==option.value))}/>{option.label}</label>)}</fieldset>:<label key={field.key}>{field.type==='checkbox'?<><input type="checkbox" checked={!!value} onChange={e=>change(e.target.checked)}/>{field.label}</>:<>{field.label}{field.type==='select'?<select aria-label={field.label} required={field.required} value={String(value)} onChange={e=>change(e.target.value)}>{field.options?.map(option=><option key={option.value} value={option.value}>{option.label}</option>)}</select>:field.type==='textarea'?<textarea aria-label={field.label} required={field.required} maxLength={field.maxLength??10000} value={String(value)} onChange={e=>change(e.target.value)}/>:<><input aria-label={field.label} type={field.type==='date'?'datetime-local':'text'} required={field.required} autoComplete="off" spellCheck={false} maxLength={field.maxLength??256} value={String(value)} list={field.type==='operator'?`${prefix}-${field.key}`:undefined} onChange={e=>change(e.target.value)}/>{field.type==='operator'&&<datalist id={`${prefix}-${field.key}`}>{field.options?.map(o=><option value={o.value} key={o.value}>{o.label}</option>)}</datalist>}</>}</>}{field.hint&&<span className="field-hint">{field.hint}</span>}</label>
 })}{error&&<p role="alert">{error}</p>}<button className="primary">Review {action.title.toLowerCase()}</button></form></details>
}
