// assets/js/mascot.js
// 吉祥物点击彩蛋
(function () {
  'use strict';

  function initMascots() {
    const mascots = document.querySelectorAll('[data-mascot]');
    mascots.forEach(function (el) {
      el.addEventListener('click', function () {
        el.classList.remove('is-poked');
        // 强制 reflow 重置 animation
        void el.offsetWidth;
        el.classList.add('is-poked');
      });
    });
  }

  window.Mascot = { initMascots: initMascots };
})();
