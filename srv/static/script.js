// 🦦 The Otter Hole Book Club - Interactive Scripts

document.addEventListener('DOMContentLoaded', function() {
  // --- Floating Sparkles ---
  const sparkles = ['✨', '⭐', '🌟', '💖', '❤️', '🦦'];
  
  function createSparkle() {
    const el = document.createElement('span');
    el.className = 'floating-sparkle';
    el.textContent = sparkles[Math.floor(Math.random() * sparkles.length)];
    el.style.left = Math.random() * 100 + 'vw';
    el.style.top = (80 + Math.random() * 20) + 'vh';
    el.style.fontSize = (0.6 + Math.random() * 1) + 'rem';
    el.style.animationDuration = (4 + Math.random() * 6) + 's';
    document.body.appendChild(el);
    el.addEventListener('animationend', () => el.remove());
  }

  // Spawn sparkles periodically
  setInterval(createSparkle, 3000);
  // Initial burst
  for (let i = 0; i < 3; i++) setTimeout(createSparkle, i * 500);

  // --- Drag-to-Rank Voting ---
  const rankings = document.getElementById('bookRankings');
  if (rankings) {
    const items = rankings.querySelectorAll('.ranking-item');
    let draggedItem = null;

    items.forEach((item, index) => {
      item.setAttribute('draggable', 'true');
      
      // Add drag handle
      const handle = document.createElement('span');
      handle.className = 'drag-handle';
      handle.textContent = '☰';
      item.insertBefore(handle, item.firstChild);

      item.addEventListener('dragstart', function(e) {
        draggedItem = this;
        this.classList.add('dragging');
        e.dataTransfer.effectAllowed = 'move';
      });

      item.addEventListener('dragend', function() {
        this.classList.remove('dragging');
        items.forEach(i => i.classList.remove('drag-over'));
        updateRankNumbers();
      });

      item.addEventListener('dragover', function(e) {
        e.preventDefault();
        e.dataTransfer.dropEffect = 'move';
        this.classList.add('drag-over');
      });

      item.addEventListener('dragleave', function() {
        this.classList.remove('drag-over');
      });

      item.addEventListener('drop', function(e) {
        e.preventDefault();
        this.classList.remove('drag-over');
        if (draggedItem !== this) {
          const allItems = [...rankings.querySelectorAll('.ranking-item')];
          const fromIdx = allItems.indexOf(draggedItem);
          const toIdx = allItems.indexOf(this);
          if (fromIdx < toIdx) {
            this.parentNode.insertBefore(draggedItem, this.nextSibling);
          } else {
            this.parentNode.insertBefore(draggedItem, this);
          }
          updateRankNumbers();
        }
      });

      // Touch support
      item.addEventListener('touchstart', function(e) {
        draggedItem = this;
        this.classList.add('dragging');
      }, {passive: true});

      item.addEventListener('touchmove', function(e) {
        e.preventDefault();
        const touch = e.touches[0];
        const target = document.elementFromPoint(touch.clientX, touch.clientY);
        if (target) {
          const targetItem = target.closest('.ranking-item');
          items.forEach(i => i.classList.remove('drag-over'));
          if (targetItem && targetItem !== draggedItem) {
            targetItem.classList.add('drag-over');
          }
        }
      });

      item.addEventListener('touchend', function(e) {
        this.classList.remove('dragging');
        const overItem = rankings.querySelector('.drag-over');
        if (overItem && draggedItem && overItem !== draggedItem) {
          const allItems = [...rankings.querySelectorAll('.ranking-item')];
          const fromIdx = allItems.indexOf(draggedItem);
          const toIdx = allItems.indexOf(overItem);
          if (fromIdx < toIdx) {
            overItem.parentNode.insertBefore(draggedItem, overItem.nextSibling);
          } else {
            overItem.parentNode.insertBefore(draggedItem, overItem);
          }
        }
        items.forEach(i => i.classList.remove('drag-over'));
        updateRankNumbers();
      });
    });

    function updateRankNumbers() {
      const currentItems = rankings.querySelectorAll('.ranking-item');
      currentItems.forEach((item, idx) => {
        const input = item.querySelector('.rank-number');
        if (input) input.value = idx + 1;
      });
    }

    // Set initial rank numbers
    updateRankNumbers();

    // Also update on manual input change
    rankings.addEventListener('input', function(e) {
      if (e.target.classList.contains('rank-number')) {
        // Validate range
        const max = items.length;
        let val = parseInt(e.target.value);
        if (val < 1) e.target.value = 1;
        if (val > max) e.target.value = max;
      }
    });
  }

  // --- Copy to clipboard for admin links ---
  document.querySelectorAll('.vote-link-box code, .admin-links code').forEach(el => {
    el.style.cursor = 'pointer';
    el.title = 'Click to copy';
    el.addEventListener('click', function() {
      navigator.clipboard.writeText(this.textContent.trim()).then(() => {
        const orig = this.textContent;
        this.textContent = 'Copied! ✨';
        setTimeout(() => this.textContent = orig, 1500);
      });
    });
  });
});
