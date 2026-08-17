// assets/js/scroll-reveal.js
// 基于 IntersectionObserver 的进入视口揭示
(function () {
  'use strict';

  function initScrollReveal(opts) {
    opts = opts || {};
    const rootMargin = opts.rootMargin || '0px 0px -10% 0px';
    const threshold = opts.threshold != null ? opts.threshold : 0.1;

    const reveals = document.querySelectorAll('.reveal');
    if (!('IntersectionObserver' in window)) {
      // 旧浏览器降级：直接全部可见
      reveals.forEach(function (el) { el.classList.add('is-visible'); });
      document.querySelectorAll('.cli-window[data-cli]').forEach(function (w) {
        const body = w.querySelector('[data-cli-lines]');
        if (body && window.Typewriter) window.Typewriter.playCliWindow(body);
      });
      return;
    }

    const observer = new IntersectionObserver(function (entries) {
      entries.forEach(function (entry) {
        if (!entry.isIntersecting) return;
        const el = entry.target;
        el.classList.add('is-visible');

        // 如果是 CLI 窗口，触发打字机
        if (el.classList.contains('cli-window')) {
          const body = el.querySelector('[data-cli-lines]');
          if (body && window.Typewriter) {
            window.Typewriter.playCliWindow(body);
          }
        }

        observer.unobserve(el);
      });
    }, { rootMargin: rootMargin, threshold: threshold });

    reveals.forEach(function (el) { observer.observe(el); });
  }

  window.ScrollReveal = { initScrollReveal: initScrollReveal };
})();
