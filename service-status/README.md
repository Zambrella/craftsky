# CraftSky public app status

A dedicated read-only Worker serves `/app.json` from private R2. Schema 2 has independent maintenance, platform minimum versions and one version-targeted message. The app presents maintenance, then required update, then an eligible undismissed message.

Set up the pinned offline validator explicitly from the repository root:

```sh
python3 -m venv service-status/.venv
service-status/.venv/bin/python -m pip install -r service-status/requirements.txt
just service-status-test
```

The CLI, local helper and test recipe all use this interpreter. `CRAFTSKY_STATUS_PYTHON` can select an existing configured environment. Validation and publication never install dependencies or retrieve schemas. Missing/broken validator or schema prevents upload.

See [the operator runbook](../docs/operations/service-status.md) for publication, clear, revision lifecycle, media hosting and release checks. `app.schema.json` is the formal structural contract; byte limits, real calendar dates, URL rules and version ranges also receive semantic validation. `contract-fixtures.json` is shared by Dart and Python.

`example.json` is an editable all-controls draft, never deployed live state. For local testing run `just service-status-dev-setup`, `just service-status-dev`, then `just app-run-status -d chrome` or a supported iOS/Android device. Edit the ignored draft and use `just service-status-dev-publish`. Omit maintenance to retain other controls; `just service-status-dev-clear` removes every control.

No hosted resource, publication or deployment is part of these tests.
