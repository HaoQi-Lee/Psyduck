// assets/js/parallax.js
// 背景网格视差（仅桌面 + 非 reduced-motion）
(function () {
  'use strict';

  function initParallax() {
    if (window.matchMedia && window.matchMedia('(pointer: coarse)').matches) return;
    if (window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;

    const targets = Array.prototype.slice.call(
      document.querySelectorAll('#hero')
    );
    if (targets.length === 0) return;

    let ticking = false;

    function update() {
      const offset = window.scrollY * 0.4;
      for (let i = 0; i < targets.length; i++) {
        targets[i].style.backgroundPositionY = (-offset) + 'px';
      }
      ticking = false;
    }

    window.addEventListener('scroll', function () {
      if (!ticking) {
        requestAnimationFrame(update);
        ticking = true;
      }
    }, { passive: true });
  }

  window.Parallax = { initParallax: initParallax };
})();
