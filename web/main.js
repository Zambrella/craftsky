/*
 * Craftsky landing page — main.js
 *
 * Responsibilities:
 *   - Open/close the waiting-list <dialog> on CTA clicks (form is a
 *     cross-origin Brevo iframe, so submission happens entirely inside it)
 *   - Open/close the AT Protocol diagram lightbox <dialog>
 *   - Dispatch PostHog events (behind DNT + key checks)
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
  const POSTHOG_HOST = 'https://t.craftsky.social';

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
      ui_host: 'https://eu.posthog.com',
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

  function openModal(source) {
    if (!modal) return;
    track('landing_cta_waiting_list_clicked', { source: source });
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

  // -----------------------------------------------------------------------
  // AT Protocol diagram lightbox — open + close-on-overlay
  // -----------------------------------------------------------------------

  const diagramModal = document.getElementById('protocol-diagram-modal');
  const diagramOpenButtons = document.querySelectorAll('.js-open-protocol-diagram');

  if (diagramModal) {
    diagramOpenButtons.forEach(function (btn) {
      btn.addEventListener('click', function () {
        track('landing_protocol_diagram_opened', { source: 'how-it-works' });
        diagramModal.showModal();
      });
    });

    // Backdrop click closes. The image is the dialog's child, so a click on
    // the dialog itself (not the image or close button) means a backdrop hit.
    diagramModal.addEventListener('click', function (event) {
      if (event.target === diagramModal) diagramModal.close();
    });
  }

  // -----------------------------------------------------------------------
  // Profile card stack
  // -----------------------------------------------------------------------

  document.querySelectorAll('[data-profile-stack]').forEach(function (stack) {
    const cards = Array.from(stack.querySelectorAll('.profile-stack__card'));
    const prevButton = stack.querySelector('[data-profile-prev]');
    const nextButton = stack.querySelector('[data-profile-next]');
    const status = stack.querySelector('[data-profile-status]');
    let activeIndex = cards.findIndex(function (card) {
      return card.getAttribute('data-active') === 'true';
    });
    let swipeStart = null;

    if (activeIndex < 0) activeIndex = 0;

    function setActive(nextIndex) {
      activeIndex = (nextIndex + cards.length) % cards.length;
      cards.forEach(function (card, index) {
        const isActive = index === activeIndex;
        card.setAttribute('data-active', isActive ? 'true' : 'false');
        card.setAttribute('aria-hidden', isActive ? 'false' : 'true');
      });
      if (status) status.textContent = (activeIndex + 1) + ' / ' + cards.length;
    }

    function showPrevious() {
      setActive(activeIndex - 1);
    }

    function showNext() {
      setActive(activeIndex + 1);
    }

    if (prevButton) prevButton.addEventListener('click', showPrevious);
    if (nextButton) nextButton.addEventListener('click', showNext);

    function getSwipePoint(event) {
      if (event.changedTouches && event.changedTouches.length > 0) {
        return {
          x: event.changedTouches[0].clientX,
          y: event.changedTouches[0].clientY,
        };
      }
      return { x: event.clientX, y: event.clientY };
    }

    function startSwipe(event) {
      if (event.button !== undefined && event.button !== 0) return;
      swipeStart = getSwipePoint(event);
    }

    function finishSwipe(event) {
      if (!swipeStart) return;
      const point = getSwipePoint(event);
      const deltaX = point.x - swipeStart.x;
      const deltaY = point.y - swipeStart.y;
      swipeStart = null;
      if (Math.abs(deltaX) < 48 || Math.abs(deltaX) < Math.abs(deltaY)) return;
      if (deltaX > 0) showPrevious();
      else showNext();
    }

    stack.addEventListener('pointerdown', startSwipe);
    stack.addEventListener('pointerup', finishSwipe);
    stack.addEventListener('mousedown', startSwipe);
    stack.addEventListener('mouseup', finishSwipe);
    stack.addEventListener('touchstart', startSwipe, { passive: true });
    stack.addEventListener('touchend', finishSwipe);

    setActive(activeIndex);
  });

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
