import { effectLabels, type Effect } from './model'
import { ConsequenceHelp } from './ConsequenceHelp'

export function EffectChoices({ legend, prefix = '', choices, value, onChange }: { legend: string; prefix?: string; choices: Effect[]; value: Effect[]; onChange: (v: Effect[]) => void }) {
  return <fieldset className="effect-choices"><legend>{legend}</legend>{choices.map(effect => <div className="effect-choice" key={effect}><label><input type="checkbox" checked={value.includes(effect)} onChange={e => onChange(e.target.checked ? [...value, effect] : value.filter(v => v !== effect))}/>{prefix}{effectLabels[effect]}</label><ConsequenceHelp effect={effect}/></div>)}</fieldset>
}
