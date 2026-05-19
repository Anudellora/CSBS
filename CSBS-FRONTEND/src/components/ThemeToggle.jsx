import { Moon, Sun } from 'lucide-react';
import { memo } from 'react';
import { useTheme } from '../context/ThemeContext';
import './ThemeToggle.css';

function ThemeToggle({ className = '' }) {
    const { isDark, toggleTheme } = useTheme();
    return (
        <button
            type="button"
            className={`theme-toggle ${className}`}
            onClick={toggleTheme}
            aria-label={isDark ? 'Переключить на светлую тему' : 'Переключить на тёмную тему'}
            title={isDark ? 'Светлая тема' : 'Тёмная тема'}
        >
            <span className={`theme-toggle-icon ${isDark ? 'is-dark' : 'is-light'}`}>
                <Sun size={18} className="theme-toggle-sun" />
                <Moon size={18} className="theme-toggle-moon" />
            </span>
        </button>
    );
}

export default memo(ThemeToggle);
