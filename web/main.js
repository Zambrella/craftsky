/*
 * Craftsky landing page — main.js
 *
 * Responsibilities:
 *   - Open/close the waiting-list <dialog> on CTA clicks
 *   - Dispatch two PostHog events (behind DNT + key checks)
 *   - Close modal on overlay click
 *   - Keep current year in footer up to date
 *
 * ESC-to-close and focus trap are native to <dialog> — no JS needed.
 */

(function () {
  'use strict';

  // -----------------------------------------------------------------------
  // Config
  // -----------------------------------------------------------------------

  const POSTHOG_KEY = 'phc_p9bFRYQRYLhWMjUFpznyKVjXLdZccJFZceEJztFuCFyv';
  const POSTHOG_HOST = 'https://eu.i.posthog.com';

  // -----------------------------------------------------------------------
  // PostHog init (DNT-aware, key-aware)
  // -----------------------------------------------------------------------

  function shouldLoadPostHog() {
    if (navigator.doNotTrack === '1') return false;
    if (POSTHOG_KEY === 'REPLACE_ME') return false;
    return true;
  }

  function loadPostHog() {
    if (!shouldLoadPostHog()) return;
    // Minimal PostHog loader. See https://posthog.com/docs/integrate/client/js
    // for the full official snippet. This trimmed version loads the script,
    // initialises with anonymous/no-cookie settings, and exposes window.posthog.
    !function (t, e) {
      var o, n, p, r;
      e.__SV ||
        ((window.posthog = e), (e._i = []),
          (e.init = function (i, s, a) {
            function g(t, e) {
              var o = e.split('.');
              2 == o.length && ((t = t[o[0]]), (e = o[1]));
              t[e] = function () { t.push([e].concat(Array.prototype.slice.call(arguments, 0))); };
            }
            (p = t.createElement('script')).type = 'text/javascript';
            p.async = !0;
            p.src = s.api_host + '/static/array.js';
            (r = t.getElementsByTagName('script')[0]).parentNode.insertBefore(p, r);
            var u = e;
            for (void 0 !== a ? (u = e[a] = []) : (a = 'posthog'),
              u.people = u.people || [],
              u.toString = function (t) { var e = 'posthog'; return 'posthog' !== a && (e += '.' + a), t || (e += ' (stub)'), e; },
              u.people.toString = function () { return u.toString(1) + '.people (stub)'; },
              o = 'capture identify alias people.set people.set_once set_config register register_once unregister opt_out_capturing has_opted_out_capturing opt_in_capturing reset isFeatureEnabled onFeatureFlags getFeatureFlag getFeatureFlagPayload reloadFeatureFlags group updateEarlyAccessFeatureEnrollment getEarlyAccessFeatures getActiveMatchingSurveys getSurveys'.split(' '), n = 0; n < o.length; n++) g(u, o[n]);
            e._i.push([i, s, a]);
          }),
          (e.__SV = 1));
    }(document, window.posthog || []);

    window.posthog.init(POSTHOG_KEY, {
      api_host: POSTHOG_HOST,
      persistence: 'memory',
      autocapture: false,
      disable_session_recording: true,
      capture_pageview: true,
      capture_pageleave: true,
    });
  }

  function track(event, properties, options) {
    if (!window.posthog || typeof window.posthog.capture !== 'function') return;
    window.posthog.capture(event, properties || {}, options);
  }

  // -----------------------------------------------------------------------
  // Modal
  // -----------------------------------------------------------------------

  const modal = document.getElementById('waiting-list-modal');
  const openButtons = document.querySelectorAll('.js-open-waiting-list');
  const form = modal && modal.querySelector('.modal__form');
  const formBody = modal && modal.querySelector('[data-state="form"]');
  const successBody = modal && modal.querySelector('[data-state="success"]');

  function openModal(source) {
    if (!modal) return;
    track('landing_cta_waiting_list_clicked', { source: source });
    if (formBody && successBody) {
      formBody.hidden = false;
      successBody.hidden = true;
    }
    modal.showModal();
  }

  openButtons.forEach(function (btn) {
    btn.addEventListener('click', function () {
      const source = btn.getAttribute('data-source') || 'unknown';
      openModal(source);
    });
  });

  // Overlay click: <dialog> clicks bubble to the dialog itself when the user
  // clicks the backdrop. Clicks on inner content have different target.
  if (modal) {
    modal.addEventListener('click', function (event) {
      if (event.target === modal) modal.close();
    });
  }

  // After the form submits, show the success state.
  // NOTE: until a real provider endpoint is set in the form action, the
  // browser will attempt to navigate to '#TODO-WAITING-LIST-ENDPOINT', which
  // means the success state won't be seen. When a provider is wired up, one
  // of two paths applies:
  //   (a) Native POST with redirect-after-submit — remove this listener.
  //   (b) Provider endpoint with no-redirect JSON response — keep this
  //       listener and preventDefault/fetch from here.
  if (form) {
    form.addEventListener('submit', function (event) {
      // Until a real provider is set, show the success state optimistically.
      const rawAction = form.getAttribute('action') || '';
      if (rawAction.startsWith('#TODO')) {
        event.preventDefault();
        if (formBody && successBody) {
          formBody.hidden = true;
          successBody.hidden = false;
        }
      }
    });
  }

  // -----------------------------------------------------------------------
  // Spec-click tracking
  // -----------------------------------------------------------------------

  document.querySelectorAll('.js-track-spec-click').forEach(function (el) {
    el.addEventListener('click', function () {
      track('landing_cta_spec_clicked', { source: 'hero' }, { transport: 'sendBeacon' });
    });
  });

  // -----------------------------------------------------------------------
  // Footer year
  // -----------------------------------------------------------------------

  const yearEl = document.getElementById('footer-year');
  if (yearEl) yearEl.textContent = new Date().getFullYear();

  // -----------------------------------------------------------------------
  // Kick off
  // -----------------------------------------------------------------------

  loadPostHog();
})();
