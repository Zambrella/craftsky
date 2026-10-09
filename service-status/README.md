# CraftSky public service status

A dedicated read-only Worker serves `/app.json` from a private R2 binding. Message updates replace the object directly; they do not deploy the website, Worker or AppView. Production and preview have separate Workers, domains and buckets.

See [the operator runbook](../docs/operations/service-status.md) for setup/change controls, contract, publish/clear commands, verification and release gaps. `example.json` is an editable draft, never a deployed live-state asset. `contract-fixtures.json` is shared by Dart and Python tests.

Run `just service-status-test` for deterministic Python/Node tests. Hosted native/browser checks are manual release checks documented in the runbook; they require separately authorized setup.

For interactive local testing: `just service-status-dev-setup`, then `just service-status-dev`; in another terminal use `just app-run-status -d chrome` (or another Flutter device). Edit the ignored local draft and run `just service-status-dev-publish`; `just service-status-dev-clear` restores normal mode. See the runbook for details.
