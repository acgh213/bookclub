// The Otter Hole Book Club

document.addEventListener('DOMContentLoaded', function() {
  // --- Subtle floating particles ---
  function createSparkle() {
    const el = document.createElement('span');
    el.className = 'floating-sparkle';
    el.textContent = '\u2727';
    el.style.left = Math.random() * 100 + 'vw';
    el.style.top = (80 + Math.random() * 20) + 'vh';
    el.style.fontSize = (0.7 + Math.random() * 0.5) + 'rem';
    el.style.animationDuration = (6 + Math.random() * 8) + 's';
    el.style.color = 'var(--lavender-deep)';
    document.body.appendChild(el);
    el.addEventListener('animationend', () => el.remove());
  }

  setInterval(createSparkle, 6000);
  setTimeout(createSparkle, 2000);

  // --- Copy to clipboard for admin links ---
  document.querySelectorAll('.vote-link-box code, .admin-links code').forEach(el => {
    el.style.cursor = 'pointer';
    el.title = 'Click to copy';
    el.addEventListener('click', function() {
      navigator.clipboard.writeText(this.textContent.trim()).then(() => {
        const orig = this.textContent;
        this.textContent = 'Copied!';
        setTimeout(() => this.textContent = orig, 1500);
      });
    });
  });
});
