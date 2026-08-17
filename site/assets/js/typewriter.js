// assets/js/typewriter.js
// 原生打字机：单元素逐字符 + CLI 多行链式
// 导出全部以全局 namespace `window.Typewriter` 提供，避免 ES Module 跨文件协议障碍。

(function () {
  'use strict';

  function sleep(ms) {
    return new Promise(function (resolve) { setTimeout(resolve, ms); });
  }

  /**
   * 逐字符打出文本到 element。
   * @param {HTMLElement} element
   * @param {{text:string, speed?:number, startDelay?:number}} opts
   */
  async function typewriterPlay(element, opts) {
    const text = opts.text || '';
    const speed = opts.speed != null ? opts.speed : 55;
    const startDelay = opts.startDelay || 0;

    element.textContent = '';
    if (startDelay > 0) await sleep(startDelay);

    for (let i = 0; i < text.length; i++) {
      element.textContent += text[i];
      await sleep(speed);
    }
  }

  /**
   * 多行链式打字（CLI demo 用）。
   * @param {HTMLElement} container - 容器，会被清空并填入若干 .cli-line
   * @param {{lines:string[], speed?:number, lineDelay?:number}} opts
   */
  async function typewriterPlayLines(container, opts) {
    const lines = opts.lines || [];
    const speed = opts.speed != null ? opts.speed : 35;
    const lineDelay = opts.lineDelay != null ? opts.lineDelay : 250;

    container.textContent = '';

    for (let i = 0; i < lines.length; i++) {
      const lineEl = document.createElement('span');
      lineEl.className = 'cli-line is-typing';
      container.appendChild(lineEl);
      await typewriterPlay(lineEl, { text: lines[i], speed: speed });
      lineEl.classList.remove('is-typing');
      if (i < lines.length - 1) await sleep(lineDelay);
    }
  }

  /**
   * 初始化页面上所有 [data-typewriter] 元素，
   * 立即开始播放（用于 Hero 标题/副标题）。
   */
  function initTypewriters(root) {
    root = root || document;
    const nodes = root.querySelectorAll('[data-typewriter]');
    nodes.forEach(function (node) {
      const text = node.getAttribute('data-typewriter') || '';
      const speed = parseInt(node.getAttribute('data-typewriter-speed') || '55', 10);
      const startDelay = parseInt(node.getAttribute('data-typewriter-delay') || '0', 10);
      typewriterPlay(node, { text: text, speed: speed, startDelay: startDelay });
    });
  }

  /**
   * 播放单个 CLI 窗口（解析 data-cli-lines JSON 数组）。
   * 已播放过的容器加 data-cli-played="1"，防止重复触发。
   */
  async function playCliWindow(element) {
    if (!element || element.dataset.cliPlayed === '1') return;
    element.dataset.cliPlayed = '1';

    let lines = [];
    try {
      lines = JSON.parse(element.getAttribute('data-cli-lines') || '[]');
    } catch (err) {
      console.warn('[typewriter] data-cli-lines 解析失败：', err);
      return;
    }
    await typewriterPlayLines(element, { lines: lines });
  }

  window.Typewriter = {
    typewriterPlay: typewriterPlay,
    typewriterPlayLines: typewriterPlayLines,
    initTypewriters: initTypewriters,
    playCliWindow: playCliWindow,
  };
})();
