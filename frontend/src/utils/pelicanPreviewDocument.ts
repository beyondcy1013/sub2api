// Results execute only inside opaque-origin sandboxed iframes. Keep generated
// HTML out of the host DOM and block network APIs before its scripts execute.
const POLICY =
  "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; img-src data: blob:; font-src data:; connect-src 'none'; frame-src 'none'; child-src 'none'; worker-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'; webrtc 'block'"

const LOCKDOWN = `(() => {
  for (const key of ['RTCPeerConnection', 'webkitRTCPeerConnection', 'mozRTCPeerConnection', 'RTCDataChannel', 'WebTransport']) {
    try { Object.defineProperty(globalThis, key, {value: undefined, writable: false, configurable: false}); } catch (_) {}
  }
  for (const key of ['alert', 'confirm', 'prompt', 'print']) {
    try { Object.defineProperty(globalThis, key, {value: () => undefined, writable: false, configurable: false}); } catch (_) {}
  }
})()`

const FIT_SCRIPT = `(() => {
  function fit() {
    try {
      const winW = window.innerWidth || document.documentElement.clientWidth || 300;
      const winH = window.innerHeight || document.documentElement.clientHeight || 200;
      if (winW <= 0 || winH <= 0) return;

      // 1. 自动检查并修复所有 SVG 元素：补齐 viewBox 与等比例适配
      const svgs = document.querySelectorAll('svg');
      svgs.forEach(svg => {
        let vb = svg.getAttribute('viewBox');
        if (!vb) {
          let w = parseFloat(svg.getAttribute('width') || '');
          let h = parseFloat(svg.getAttribute('height') || '');
          if ((!w || !h) && svg.getBBox) {
            try {
              const bbox = svg.getBBox();
              if (bbox && bbox.width > 0 && bbox.height > 0) {
                w = bbox.width;
                h = bbox.height;
                vb = \`\${bbox.x || 0} \${bbox.y || 0} \${bbox.width} \${bbox.height}\`;
              }
            } catch (_) {}
          }
          if (!vb && w > 0 && h > 0) {
            vb = \`0 0 \${w} \${h}\`;
          }
          if (vb) {
            svg.setAttribute('viewBox', vb);
          }
        }
        svg.setAttribute('preserveAspectRatio', 'xMidYMid meet');
        svg.style.maxWidth = '100%';
        svg.style.maxHeight = '100%';
        svg.style.objectFit = 'contain';
      });

      // 2. 检查是否有固定大尺寸的父容器或内容超出视口，整体等比例居中缩小
      const body = document.body;
      const rootEl = document.documentElement;
      if (!body) return;

      body.style.transform = '';
      body.style.transformOrigin = 'center center';

      const contentW = Math.max(body.scrollWidth, rootEl.scrollWidth);
      const contentH = Math.max(body.scrollHeight, rootEl.scrollHeight);

      if (contentW > winW + 2 || contentH > winH + 2) {
        const scale = Math.min((winW - 8) / contentW, (winH - 8) / contentH);
        if (scale > 0 && scale < 0.999) {
          body.style.transform = \`scale(\${scale})\`;
          body.style.transformOrigin = 'center center';
        }
      }
    } catch (_) {}
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', fit);
  } else {
    fit();
  }
  window.addEventListener('load', fit);
  window.addEventListener('resize', fit);
  setTimeout(fit, 80);
  setTimeout(fit, 300);
  setTimeout(fit, 1000);
})()`

const FIT_STYLE = `
  html, body {
    margin: 0 !important;
    padding: 0 !important;
    width: 100% !important;
    height: 100% !important;
    overflow: hidden !important;
    display: flex !important;
    align-items: center !important;
    justify-content: center !important;
    background: transparent !important;
    box-sizing: border-box !important;
  }
  svg {
    max-width: 100% !important;
    max-height: 100% !important;
    width: auto !important;
    height: auto !important;
    object-fit: contain !important;
    display: block !important;
  }
`

export function pelicanPreviewDocument(html: string): string {
  return `<meta http-equiv="Content-Security-Policy" content="${POLICY}"><meta name="referrer" content="no-referrer"><meta name="viewport" content="width=device-width, initial-scale=1.0"><style>${FIT_STYLE}</style><script>${LOCKDOWN}</script><script>${FIT_SCRIPT}</script>${html}`
}
