// Run before styles and page content so the saved theme applies on first paint.
try {
	const theme = localStorage.getItem('craftsky.theme');
	if (theme === 'light' || theme === 'dark') {
		document.documentElement.dataset.theme = theme;
	}
} catch {
	// Storage may be unavailable; CSS follows the system preference.
}
