// Package testpipeline is DISPOSABLE. It exists to validate the appview's
// firehose → index → Postgres → HTTP read API loop end to end using the
// throwaway lexicon social.craftsky.test.post and the test_posts table.
//
// When the real social.craftsky.feed.post indexer lands, DELETE the
// entire package (rm -rf appview/internal/testpipeline/) along with:
//
//   - lexicon/social/craftsky/test/post.json
//   - appview/migrations/000004_test_posts.up.sql + .down.sql
//     (plus a follow-up drop migration)
//   - the GET /test/feed route registration in internal/routes/routes.go
//   - the Dispatcher.Register call for social.craftsky.test.post in
//     internal/app/deps.go
//
// The Dispatcher itself stays.
//
// See docs/superpowers/specs/2026-04-19-test-pipeline-design.md.
package testpipeline
