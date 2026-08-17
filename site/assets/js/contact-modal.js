// assets/js/contact-modal.js
// Contact 二维码弹窗：点击 CTA 触发；Esc / 背景点击 / 关闭按钮 关闭
// 触发器用 <a href="assets/qrcode.jpg"> —— 无 JS 时降级为新标签页打开图片
(function () {
  'use strict';

  var SELECTOR_TRIGGER = '[data-contact-trigger]';
  var SELECTOR_CLOSE = '[data-contact-close]';
  var OPEN_CLASS = 'is-open';

  var modal = document.getElementById('contact-modal');
  var lastFocused = null;

  function open() {
    if (!modal) return;
    lastFocused = document.activeElement;
    modal.classList.add(OPEN_CLASS);
    modal.setAttribute('aria-hidden', 'false');
    document.body.style.overflow = 'hidden';
    document.addEventListener('keydown', onKeydown);
    var closeBtn = modal.querySelector('.modal-close');
    if (closeBtn) closeBtn.focus();
  }

  function close() {
    if (!modal) return;
    modal.classList.remove(OPEN_CLASS);
    modal.setAttribute('aria-hidden', 'true');
    document.body.style.overflow = '';
    document.removeEventListener('keydown', onKeydown);
    if (lastFocused && typeof lastFocused.focus === 'function') lastFocused.focus();
  }

  function onKeydown(e) {
    if (e.key === 'Escape') close();
  }

  function initContactModal() {
    if (!modal) return;

    document.querySelectorAll(SELECTOR_TRIGGER).forEach(function (trigger) {
      trigger.addEventListener('click', function (e) {
        e.preventDefault();
        open();
      });
    });

    modal.querySelectorAll(SELECTOR_CLOSE).forEach(function (el) {
      el.addEventListener('click', close);
    });
  }

  window.ContactModal = {
    initContactModal: initContactModal,
    open: open,
    close: close,
  };
})();
