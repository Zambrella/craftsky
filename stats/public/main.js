(() => {
  'use strict';

  const content = document.querySelector('#stats-content');
  const chart = document.querySelector('#craft-chart');
  const status = document.querySelector('#data-status');
  const errorNote = document.querySelector('#error-note');
  const snapshot = document.querySelector('#snapshot-time');
  const fields = ['accounts', 'posts', 'projectPosts', 'finishedProjectPosts'];
  const format = new Intl.NumberFormat('en-GB');
  const motion = window.matchMedia('(prefers-reduced-motion: reduce)');
  const prefix = 'social.craftsky.feed.defs#';
  const crafts = {
    knitting: { label: 'Knitting', color: '#736197', icon: '/craft_icons/knitting.svg' },
    crochet: { label: 'Crochet', color: '#bd684c', icon: '/craft_icons/crochet.svg' },
    sewing: { label: 'Sewing', color: '#638363', icon: '/craft_icons/sewing.svg' },
    embroidery: { label: 'Embroidery', color: '#a98538', icon: '/craft_icons/embroidery.svg' },
    quilting: { label: 'Quilting', color: '#597f96', icon: '/craft_icons/quilting.svg' },
  };
  let request;
  const animations = new Set();

  function cancelAnimations() {
    animations.forEach(cancelAnimationFrame);
    animations.clear();
  }

  function isCount(value) {
    return Number.isSafeInteger(value) && value >= 0;
  }

  function validate(data) {
    if (!data || !fields.every(field => isCount(data[field])) ||
        typeof data.generatedAt !== 'string' || !Number.isFinite(Date.parse(data.generatedAt)) ||
        !Array.isArray(data.projectsByCraft) || data.projectsByCraft.length > 250 ||
        data.finishedProjectPosts > data.projectPosts || data.projectPosts > data.posts) {
      throw new Error('Invalid statistics response');
    }
    const seen = new Set();
    let total = 0;
    for (const row of data.projectsByCraft) {
      if (!row || typeof row.craft !== 'string' || row.craft.length === 0 || row.craft.length > 200 ||
          !isCount(row.count) || seen.has(row.craft)) throw new Error('Invalid craft statistics');
      seen.add(row.craft);
      total += row.count;
    }
    if (!Number.isSafeInteger(total) || total !== data.projectPosts) throw new Error('Invalid project total');
    return data;
  }

  function countUp(element, target) {
    if (motion.matches || target === 0) {
      element.textContent = format.format(target);
      return;
    }
    const start = performance.now();
    let frame;
    function step(now) {
      animations.delete(frame);
      const progress = Math.min((now - start) / 1200, 1);
      element.textContent = format.format(Math.round(target * (1 - (1 - progress) ** 3)));
      if (progress < 1 && !motion.matches) {
        frame = requestAnimationFrame(step);
        animations.add(frame);
      } else {
        element.textContent = format.format(target);
      }
    }
    frame = requestAnimationFrame(step);
    animations.add(frame);
  }

  function node(tag, className, text) {
    const element = document.createElement(tag);
    if (className) element.className = className;
    if (text !== undefined) element.textContent = text;
    return element;
  }

  function craftInfo(id) {
    if (id.startsWith(prefix) && Object.hasOwn(crafts, id.slice(prefix.length))) return crafts[id.slice(prefix.length)];
    // Unknown tokens remain readable without inserting server-provided markup.
    const token = id.slice(id.lastIndexOf('#') + 1).replace(/([a-z])([A-Z])/g, '$1 $2').replace(/[-_]/g, ' ');
    return { label: token.charAt(0).toUpperCase() + token.slice(1), color: '#777b64', mark: '✧' };
  }

  function chartMessage(message) {
    chart.replaceChildren(node('p', 'chart-message', message));
  }

  function drawChart(data) {
    if (data.projectPosts === 0) {
      chartMessage('No project posts yet.');
      return;
    }
    const rows = new Map(Object.keys(crafts).map(key => [prefix + key, 0]));
    data.projectsByCraft.forEach(row => rows.set(row.craft, row.count));
    const sorted = [...rows].sort((a, b) => b[1] - a[1] || craftInfo(a[0]).label.localeCompare(craftInfo(b[0]).label));
    const maximum = sorted[0][1];
    const magnitude = 10 ** Math.floor(Math.log10(maximum));
    const axisMax = Math.ceil(maximum / magnitude) * magnitude;
    const list = node('ul', 'craft-list');
    list.setAttribute('aria-label', 'Project posts by primary craft');
    sorted.forEach(([craft, count], index) => {
      const info = craftInfo(craft);
      const row = node('li', 'craft-row');
      row.dataset.craft = craft;
      row.style.setProperty('--bar-color', info.color);
      row.style.setProperty('--bar-width', `${count / axisMax * 100}%`);
      row.style.setProperty('--delay', `${Math.min(index, 10) * 85}ms`);
      const label = node('span', 'craft-label');
      const mark = node('span', 'craft-mark', info.mark);
      mark.setAttribute('aria-hidden', 'true');
      // Only fixed local app assets become CSS URLs; unknown craft labels
      // retain their text fallback without constructing a resource path.
      if (info.icon) {
        mark.dataset.icon = info.icon.split('/').pop();
        mark.style.setProperty('--craft-icon', `url("${info.icon}")`);
      }
      label.append(mark, node('span', '', info.label));
      const track = node('div', 'bar-track');
      track.setAttribute('aria-hidden', 'true');
      track.append(node('div', 'bar-fill'));
      row.append(label, track, node('span', 'craft-count', format.format(count)));
      list.append(row);
    });
    const axis = node('div', 'chart-axis');
    axis.setAttribute('aria-hidden', 'true');
    const ticks = node('div', 'chart-ticks');
    // With very small counts, only show meaningful integer ticks.
    const quarters = axisMax >= 4 ? [0, .25, .5, .75, 1] : [0, 1];
    quarters.forEach(part => ticks.append(node('span', '', format.format(Math.round(axisMax * part)))));
    axis.append(node('span'), ticks, node('span'));
    chart.replaceChildren(list, axis);
  }

  async function load() {
    const restoreFocus = document.activeElement === document.querySelector('#retry');
    request?.abort();
    const controller = new AbortController();
    request = controller;
    cancelAnimations();
    content.dataset.state = 'loading';
    content.setAttribute('aria-busy', 'true');
    chart.setAttribute('aria-busy', 'true');
    status.textContent = 'Loading…';
    errorNote.hidden = true;
    fields.forEach(field => {
      const element = document.querySelector(`[data-stat="${field}"]`);
      element.textContent = '—';
      element.setAttribute('aria-label', 'Loading');
    });
    chartMessage('Loading…');
    const timeout = setTimeout(() => controller.abort(), 12000);
    try {
      const endpoint = document.querySelector('meta[name="craftsky-stats-api"]').content;
      const response = await fetch(endpoint, {
        credentials: 'omit', cache: 'no-store', mode: 'cors',
        referrerPolicy: 'no-referrer', signal: controller.signal,
        headers: { Accept: 'application/json' },
      });
      if (!response.ok) throw new Error('Statistics unavailable');
      const data = validate(await response.json());
      if (request !== controller) return;
      fields.forEach(field => {
        const element = document.querySelector(`[data-stat="${field}"]`);
        element.removeAttribute('aria-label');
        countUp(element, data[field]);
      });
      drawChart(data);
      const date = new Date(data.generatedAt);
      const timestamp = new Intl.DateTimeFormat('en-GB', { dateStyle: 'medium', timeStyle: 'short', timeZone: 'UTC' }).format(date);
      snapshot.replaceChildren(document.createTextNode('Updated '));
      const time = node('time', '', `${timestamp} UTC`);
      time.dateTime = date.toISOString();
      snapshot.append(time);
      status.textContent = 'All time';
      content.dataset.state = 'ready';
      if (restoreFocus) document.querySelector('#headline-numbers').focus({ preventScroll: true });
    } catch {
      if (request !== controller) return;
      content.dataset.state = 'error';
      status.textContent = 'Unavailable';
      errorNote.hidden = false;
      fields.forEach(field => document.querySelector(`[data-stat="${field}"]`).setAttribute('aria-label', 'Unavailable'));
      chartMessage('Chart unavailable.');
      snapshot.textContent = 'Statistics unavailable.';
    } finally {
      clearTimeout(timeout);
      if (request === controller) {
        content.setAttribute('aria-busy', 'false');
        chart.setAttribute('aria-busy', 'false');
      }
    }
  }

  document.querySelector('#retry').addEventListener('click', load);
  load();
})();
