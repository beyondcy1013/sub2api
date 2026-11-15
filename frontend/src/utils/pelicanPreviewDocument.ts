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
    width: 100% !important;
    height: auto !important;
    object-fit: contain !important;
    display: block !important;
  }
`

// 预览 iframe 使用 srcdoc，会继承宿主文档的 CSP；宿主 script-src 只放行 nonce，
// 因此内联脚本必须携带同一 nonce，否则动画脚本会被静默拦截，预览只剩首帧。
export function currentCspNonce(): string {
  try {
    const el = document.querySelector('script[nonce]') as HTMLScriptElement | null
    return el?.nonce || ''
  } catch {
    return ''
  }
}

function escapeAttr(value: string): string {
  return value.replace(/&/g, '&amp;').replace(/"/g, '&quot;')
}

function withScriptNonce(html: string, nonce: string): string {
  if (!nonce) return html
  return html.replace(/<script\b([^>]*)>/gi, (_match, attrs: string) => {
    const cleaned = attrs.replace(/\snonce\s*=\s*("[^"]*"|'[^']*'|[^\s>]+)/gi, '')
    return `<script${cleaned} nonce="${escapeAttr(nonce)}">`
  })
}

export function pelicanPreviewDocument(html: string, nonce: string = currentCspNonce()): string {
  const nonceAttr = nonce ? ` nonce="${escapeAttr(nonce)}"` : ''
  return `<meta http-equiv="Content-Security-Policy" content="${POLICY}"><meta name="referrer" content="no-referrer"><meta name="viewport" content="width=device-width, initial-scale=1.0"><style>${FIT_STYLE}</style><script${nonceAttr}>${LOCKDOWN}</script><script${nonceAttr}>${FIT_SCRIPT}</script>${withScriptNonce(html, nonce)}`
}

// Older history entries may contain only raw model output, without result metadata.
export function extractPelicanPreviewHtml(output: string): string {
  const fenced = /```(?:html|svg|xml)?\s*\n([\s\S]*?)```/gi
  for (const match of output.matchAll(fenced)) {
    const html = extractPelicanPreviewHtml(match[1])
    if (html) return html
  }
  return output.match(/(?:<!doctype\s+html[^>]*>\s*)?<html\b[\s\S]*?<\/html\s*>/i)?.[0]
    || output.match(/<svg\b[\s\S]*?<\/svg\s*>/i)?.[0]
    || ''
}
