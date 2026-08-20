document.querySelectorAll('[data-reveal]').forEach((button) => {
  button.addEventListener('click', () => {
    const answer = document.getElementById(button.dataset.reveal);
    answer.hidden = false;
    button.disabled = true;
    answer.focus?.();
  });
});
