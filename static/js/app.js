document.addEventListener('DOMContentLoaded', () => {
    const themeToggle = document.getElementById('theme-toggle');
    const themeIcon = document.getElementById('theme-icon');
    const body = document.body;

    // Funkcja do ustawiania motywu
    const setTheme = (theme) => {
        if (theme === 'dark') {
            body.classList.add('dark-theme');
            themeIcon.textContent = '☀️'; // Ikona słońca dla ciemnego motywu
            localStorage.setItem('theme', 'dark');
        } else {
            body.classList.remove('dark-theme');
            themeIcon.textContent = '🌙'; // Ikona księżyca dla jasnego motywu
            localStorage.setItem('theme', 'light');
        }
    };

    // Sprawdź, czy użytkownik ma zapisany motyw w Local Storage
    const savedTheme = localStorage.getItem('theme');
    if (savedTheme) {
        setTheme(savedTheme);
    } else if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
        // Jeśli nie ma zapisanego motywu, sprawdź preferencje systemowe
        setTheme('dark');
    } else {
        // Domyślnie ustaw jasny motyw
        setTheme('light');
    }

    // Obsługa kliknięcia przycisku
    if (themeToggle) {
        themeToggle.addEventListener('click', () => {
            if (body.classList.contains('dark-theme')) {
                setTheme('light');
            } else {
                setTheme('dark');
            }
        });
    }
});