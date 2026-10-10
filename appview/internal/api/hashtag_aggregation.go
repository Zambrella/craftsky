package api

// hashtagAggregationCTEs follows an eligible CTE with uri, tag_key, and
// tag_spellings columns. Counts and spelling votes use that same snapshot;
// overlapping spellings never inflate the aggregate distinct-post count.
const hashtagAggregationCTEs = `,
 counts AS (
  SELECT tag_key, count(DISTINCT uri)::int AS post_count
  FROM eligible GROUP BY tag_key
 ),
 spelling_counts AS (
  SELECT e.tag_key, spelling COLLATE "C" AS spelling, count(DISTINCT e.uri) AS frequency
  FROM eligible e CROSS JOIN LATERAL unnest(e.tag_spellings) AS spelling
  WHERE craftsky_tag_lower(spelling) = e.tag_key
  GROUP BY e.tag_key, spelling COLLATE "C"
 ),
 winners AS (
  SELECT DISTINCT ON (tag_key) tag_key, spelling
  FROM spelling_counts ORDER BY tag_key, frequency DESC, spelling COLLATE "C" ASC
 )
 SELECT w.spelling, c.post_count FROM counts c JOIN winners w USING (tag_key)
`
