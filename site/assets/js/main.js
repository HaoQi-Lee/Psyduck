// assets/js/main.js
// 入口：按依赖顺序初始化各模块
(function () {
  'use strict';

  function boot() {
    // 1. 打字机（Hero 立即开始）
    if (window.Typewriter) window.Typewriter.initTypewriters();

    // 2. 滚动揭示（同时托管 CLI 打字机的触发）
    if (window.ScrollReveal) window.ScrollReveal.initScrollReveal();

    // 3. 鼠标光晕（仅细指针）
    if (window.CursorGlow) window.CursorGlow.initCursorGlow();

    // 4. 视差（仅桌面 + 非 reduced-motion）
    if (window.Parallax) window.Parallax.initParallax();

    // 5. 吉祥物彩蛋
    if (window.Mascot) window.Mascot.initMascots();

    // 6. Contact 二维码弹窗
    if (window.ContactModal) window.ContactModal.initContactModal();
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', boot);
  } else {
    boot();
  }
})();
