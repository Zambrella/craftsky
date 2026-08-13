(function () {
  'use strict';

  const successMessage = document.getElementById('success-message');
  if (!successMessage) return;

  function redirectAfterSuccess() {
    if (window.getComputedStyle(successMessage).display !== 'none') {
      window.location.replace('/');
    }
  }

  new MutationObserver(redirectAfterSuccess).observe(successMessage, {
    attributes: true,
    attributeFilter: ['class', 'style'],
  });
})();
