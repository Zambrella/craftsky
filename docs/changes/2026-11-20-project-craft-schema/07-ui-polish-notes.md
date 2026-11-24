# Project pattern editor polish

## Simplified composer and filter layout

Use Pattern name in both composer surfaces and preserve hashtag affordances, pattern rows, and their existing actions. Keep the shared field sizing and focus styling. Let the selected project type reveal its listed subtypes. Other reveals a text field. Craft method, style, and equipment choices remain independent of the item list. The authoring and filter surfaces use the shorter field inventory in the [requirements](01-requirements.md#approved-simplification).

## Alternative project item names

The Flutter and web project type and subtype dropdowns show alternate names below the main option in the existing secondary text style. For example, **Dungarees** shows **Overalls** and **Trousers** shows **Pants**. Skip aliases already present in the label, and keep all catalogue terms available to picker search. Selected values show their main labels.

Six web browser tests cover composer and filter dropdowns, accessibility in both themes, and method alias search. The production build, Svelte diagnostics, lint, and 635 unit tests passed. Flutter analysis and 37 focused widget tests passed.

## Consistent display for older projects

Project cards and post details in both clients use the current field inventory. Feed cards retain their compact pattern summaries. Post details retain pattern links and annotations, tags, and current fields for the selected craft. Retired structured details such as materials, colour and design tags, duration, sizes, gauge, and technique groups stay in the saved record.

Five web browser regressions passed, including the older project feed card, pattern links and credits, and the supported Crochet method on older records. The web production build, lint, Svelte diagnostics, and 634 unit tests passed. The three Flutter project card, feed card, and post detail suites passed 149 tests.

## Flat craft filter form

Both clients show status, pattern difficulty, self-drafted, and the fields available for the selected craft in a single list. Each picker opens directly from its field. Project subtype stays visible and disabled until a selected project type provides choices.

Thirty web browser tests and 38 Flutter project page and filter tests passed. Verification covered field order, dependent selections, Apply, Clear all, and navigation. The web production build, web unit tests, and both clients’ static checks passed. The iPhone debug app also hot-reloaded successfully.

## Alphabetical options and custom names under Other

Project composer and craft filter dropdowns use display labels for alphabetical ordering. Other appears last wherever it is offered and provides the optional custom item name in the composer. Primary craft uses the same order as Home tabs, with saved profile interests first. Visible alternative names begin with a capital and appear in alphabetical order. A previously saved custom subtype under a named type stays visible until it is cleared or replaced.

The browser run passed 36 tests covering dropdowns, accessibility, and saving an Other name while preserving earlier named values. Web static checks, the production build, and 634 unit tests passed. Flutter analysis passed, and 309 tests passed with three conditional scheduling skips. The iPhone debug app hot-reloaded successfully.

## Short labels for equivalent names

Knitting and Crochet show **Jumper** with **Pullover** underneath. Sewing and Embroidery show **Repair** with **Mending** underneath. Both names and the former combined labels remain searchable. Keep combined labels where the option covers distinct items, such as **Coat or jacket**. Existing records and filters use the same catalogue identifiers.

The shared catalogue and AppView cursor tests passed. The web project and filter suites passed 18 browser tests, the generated schema preview passed four checks, and Flutter passed 41 focused tests. The iPhone debug app hot-reloaded successfully.

## Project composer field order

The main composer starts with media and Primary craft. It then shows the craft details and Pattern details entries, followed by Status, Project title, and Project description. Additional crafts opens with the selected craft's details, including for Spinning. Pattern names and the other pattern fields open together in Pattern details. Both pages open directly from the main form, and the Add more details section is removed.

The web composer groups Craft details and Pattern details in borderless rows with a 4px gap.

The web build, static checks, and 634 unit tests passed. The 107 browser cases and three final smoke checks covered the new order, both detail pages, pattern hashtags, draft reopening, scheduling, and edits to published posts. Flutter analysis and 161 widget tests passed. The three conditional scheduling tests passed in a separate subscriptions-enabled run, and the iPhone debug app hot-reloaded successfully.

## Additional crafts and required status

Additional crafts comes after the selected craft's own fields. Status starts at Finished on new projects. Authors can switch status using the picker. Saving a published edit, scheduling, or publishing requires a value. An older project without a status prompts the author to choose one. Private drafts can still be saved with an empty status.

The latest field and option ordering checks passed in the 91-test browser run and the 203-test Flutter run. The three opt-in Flutter scheduling cases also passed. The full subscription-enabled scheduling run still has the existing free-tier paywall test timeout.

## Composer hints and mobile Done action

The project description placeholder explains mentions, hashtags, and links in a second sentence. Both sentences end with a full stop and are separated by a blank line. Each pattern's Designer and Publisher placeholders explain that authors can enter a name or use a mention.

Regular posts show the same mentions, hashtags, and links guidance below their opening placeholder on mobile and web.

The regular-post hint passed 28 Flutter tests and four browser tests, including checks of the reply, quote, and project prompts.

On mobile, the shared Craft details page and Pattern details use the main composer's floating bottom action. Done stays in place while the fields scroll. It dismisses the keyboard while typing and returns to the composer when the keyboard is closed.

All 21 browser tests and 27 Flutter tests passed. Web build and static checks, Flutter analysis, and iPhone hot reload passed.

## Short craft method labels

Crochet shows Standard and Tunisian, Knitting shows Loom, and Felting shows Wet. The shorter labels appear in the composer, craft filters, project displays, and schema preview. Picker search accepts the full names and existing aliases.

Seven web browser checks, four schema preview tests, 636 web unit tests, 64 Flutter tests, and the AppView catalogue tests passed. Web lint and type checks, Flutter analysis, and iPhone hot reload passed.

## Pattern details by craft

Spinning and Dyeing show Pattern details when an additional craft uses patterns or the project already has pattern information. Weaving labels the section Pattern or draft, and Felting uses Pattern or template. The name field follows the same wording on every row. The existing pattern fields, links, and hashtags support all three labels.

Verification passed with 636 web unit tests, 27 focused browser tests on the final build, 73 Flutter composer tests, and a subscription-enabled scheduled Dyeing test. Both clients passed static checks, and the iPhone debug app hot-reloaded.

## Project item group and summary

Craft details shows the Project type picker and any Other name field first, followed by craft-specific options and Additional crafts. The Craft details row in the main composer shows the saved or selected labels, such as Garment · Jumper.

The final flat layout passed 24 browser tests and 24 Flutter tests, including field order, selection summaries, and custom names. Both clients passed static checks, and the iPhone debug app hot-reloaded.

## Combined Project type picker

The composer and craft filters show the full catalogue in one searchable list. When the list includes subtypes, all top-level options use bold labels, including standalone types such as Costume. Subtypes are indented, and a small gap separates groups. Selecting a type alone saves that type. Selecting a subtype saves both values and displays both labels. Other remains last and offers a custom name in the composer.

The filter sends each choice as a repeatable `projectItem` parameter. A project can match any selected type or subtype and must match the other active filters. Existing links with `projectType` and `projectSubtype` retain their original intersection until the item selection changes. The record format and database schema stay the same.

The revised group spacing passed 162 Flutter tests and four browser tests, including the composer, filters, and keyboard navigation in both web themes. Static checks passed and the iPhone simulator hot-reloaded.

## Earlier UI polish

## Summary

The Flutter and web composers use the same hierarchy and actions when an author adds a second pattern. This pass covers the pattern editor in AT-009 and the existing scheduled edit flow in AT-008.

## Polish items

| Request | Change |
|---|---|
| Pattern headings look smaller than other text | Numbered headings use title typography on Flutter and 20px on web. |
| Dividers belong between patterns | Each numbered row after Pattern 1 has a divider above it. |
| The Remove pattern button stays visible with one row | The row removal control is visible only when the editor contains multiple rows. |
| Remove pattern needs destructive styling and an icon | The label and bin icon both use the theme's error colour. |
| The add action needs clearer copy and an icon | The action reads **Add another pattern** and uses a plus icon. |
| Additional patterns need the same name and Self-drafted cues as Pattern 1 | Every blank pattern name field shows `#PatternName` as a placeholder on mobile and web. Each Self-drafted switch shows “I drafted this pattern.” |
| Web Pattern details should match the Flutter action layout | Remove pattern sits next to its numbered heading with a red icon and label. Add another pattern sits below the final entry with a primary-colour icon and label. Both web actions use flat text buttons. |

## Verification

- Flutter analysis and eight composer widget tests passed. The subscription-enabled AT-008 schedule test passed through save, reopen and Post now.
- The Flutter bin colour fix passed all 11 composer widget tests. The open iPhone simulator showed a red bin beside the red Remove pattern label after hot reload.
- Divider placement passed 18 Flutter tests and two browser tests, including the transition between single and multiple patterns.
- The pattern-name placeholder passed 102 Flutter widget and localisations tests and four browser tests. Coverage includes empty rows, craft-specific labels, reopening a plain or hashtagged name, and submitting a pattern that has a link and a blank name.
- The web action layout passed 12 browser tests, including heading alignment, icon colours, keyboard access, and pattern persistence through draft, published, and scheduled editing.
- Svelte diagnostics, targeted lint and the production build passed. Five local browser tests passed. They check the two-to-one transition, the remaining designer credit, divider placement and action styling, and exercise draft, published and scheduled editing.

## Scope

The polish changes affect presentation and when each row offers its removal control. The author can edit a sole pattern through its fields. Record validation and serialization follow the approved pattern contract.

## Web field alignment

The web composer uses the same 55px resting height, horizontal inset and label spacing for empty pickers and single-line fields. Search and clear icons line up with the field padding. Placeholders use the shared muted colour, and focused controls share an inset outline and theme-aware shadow. Multi-select fields grow to fit selected chips, short lists keep their chevrons, and dependent pickers retain their disabled state.

Gauge fields share label and input rows so wrapped labels do not move or stretch the controls next to them. Browser checks compared the main composer and Crochet detail fields in the four-column light layout at 1280px and the two-column dark layout at 402px. All 69 browser tests passed across composer details, input parity, filter details and UI polish. The production build, Svelte diagnostics and targeted lint and formatting checks passed.
