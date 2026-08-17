// assets/js/cursor-glow.js
// 鼠标跟随光晕（仅细指针设备）
(function () {
  'use strict';

  function initCursorGlow() {
    if (window.matchMedia && window.matchMedia('(pointer: coarse)').matches) {
      return; // 移动端关闭
    }

    const glow = document.createElement('div');
    glow.className = 'cursor-glow is-hidden';
    glow.setAttribute('aria-hidden', 'true');
    document.body.appendChild(glow);

    let x = 0, y = 0, ticking = false;

    function update() {
      glow.style.transform = 'translate3d(' + (x - 200) + 'px,' + (y - 200) + 'px, 0)';
      ticking = false;
    }

    window.addEventListener('mousemove', function (e) {
      x = e.clientX;
      y = e.clientY;
      if (glow.classList.contains('is-hidden')) glow.classList.remove('is-hidden');
      if (!ticking) {
        requestAnimationFrame(update);
        ticking = true;
      }
    }, { passive: true });

    window.addEventListener('mouseleave', function () {
      glow.classList.add('is-hidden');
    });
  }

  window.CursorGlow = { initCursorGlow: initCursorGlow };
})();
