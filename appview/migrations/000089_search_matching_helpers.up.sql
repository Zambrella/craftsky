-- Query-local search helpers. No stored projection or record rewrites.
CREATE FUNCTION craftsky_search_score(fields text[], weights real[], english boolean, concepts jsonb)
RETURNS double precision LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
DECLARE
    concept jsonb;
    alternative text;
    config regconfig;
    query tsquery;
    document tsvector;
    best double precision;
    total double precision := 0;
    i integer;
BEGIN
    IF jsonb_array_length(concepts) = 0 THEN RETURN NULL; END IF;
    FOR concept IN SELECT value FROM jsonb_array_elements(concepts) LOOP
        best := NULL;
        FOR alternative IN SELECT jsonb_array_elements_text(concept) LOOP
            FOREACH config IN ARRAY CASE WHEN english THEN ARRAY['simple'::regconfig, 'english'::regconfig] ELSE ARRAY['simple'::regconfig] END LOOP
                query := phraseto_tsquery(config, alternative);
                IF numnode(query) = 0 THEN CONTINUE; END IF;
                FOR i IN 1..coalesce(array_length(fields, 1), 0) LOOP
                    document := to_tsvector(config, coalesce(fields[i], ''));
                    IF document @@ query THEN
                        best := greatest(best, ts_rank_cd(document, query)::double precision * weights[i]);
                    END IF;
                END LOOP;
            END LOOP;
        END LOOP;
        IF best IS NULL THEN RETURN NULL; END IF;
        total := total + best;
    END LOOP;
    RETURN total;
END;
$$;

-- Exactly one insertion, deletion, substitution or adjacent transposition.
-- Character offsets are Unicode character offsets, not byte offsets.
CREATE FUNCTION craftsky_search_one_edit(a text, b text)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE STRICT AS $$
DECLARE
    na integer := char_length(a);
    nb integer := char_length(b);
    i integer := 1;
BEGIN
    IF a = b OR abs(na - nb) > 1 THEN RETURN false; END IF;
    WHILE i <= least(na, nb) AND substr(a, i, 1) = substr(b, i, 1) LOOP
        i := i + 1;
    END LOOP;
    IF na = nb THEN
        RETURN substr(a, i + 1) = substr(b, i + 1)
            OR (i < na AND substr(a, i, 1) = substr(b, i + 1, 1)
                AND substr(a, i + 1, 1) = substr(b, i, 1)
                AND substr(a, i + 2) = substr(b, i + 2));
    ELSIF na > nb THEN
        RETURN substr(a, i + 1) = substr(b, i);
    ELSE
        RETURN substr(a, i) = substr(b, i + 1);
    END IF;
END;
$$;
