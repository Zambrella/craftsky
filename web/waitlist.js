(function () {
  'use strict';

  const successMessage = document.getElementById('success-message');
  if (!successMessage) return;

  function redirectAfterSuccess() {
    if (successMessage.classList.contains('sib-form-message-panel--active')) {
      window.location.replace('/');
    }
  }

  new MutationObserver(redirectAfterSuccess).observe(successMessage, {
    attributes: true,
    attributeFilter: ['class', 'style'],
  });
})();
